package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
)

const errorEventType = ""

// captureTransport collects every event the Sentry client would otherwise ship
// over the network, so tests can assert against them in-process. It is safe for
// concurrent use because the Sentry hub may call SendEvent from multiple
// goroutines during transaction.Finish + recoverWithSentry interleaving.
//
// [Ja] Sentry クライアントが本来ネットワーク送信するイベントをすべて収集する
// テスト用 Transport。transaction.Finish と recoverWithSentry が別ゴルーチンから
// SendEvent を呼ぶ可能性があるため排他制御で守る。
type captureTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *captureTransport) Configure(_ sentry.ClientOptions)        {}
func (t *captureTransport) Flush(_ time.Duration) bool              { return true }
func (t *captureTransport) FlushWithContext(_ context.Context) bool { return true }
func (t *captureTransport) Close()                                  {}
func (t *captureTransport) SendEventWithContext(_ context.Context, e *sentry.Event) {
	t.SendEvent(e)
}

func (t *captureTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *captureTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*sentry.Event, len(t.events))
	copy(out, t.events)
	return out
}

// newTestHub builds a per-test Sentry Hub backed by a captureTransport so each
// test sees its own isolated event stream and we never touch the global hub.
//
// [Ja] テストごとに独立した Hub + captureTransport を作る。グローバル Hub には
// 一切触らない。
func newTestHub(t *testing.T) (*sentry.Hub, *captureTransport) {
	t.Helper()
	transport := &captureTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:              "https://public@example.com/1",
		Transport:        transport,
		EnableTracing:    true,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		t.Fatalf("sentry.NewClient: %v", err)
	}
	return sentry.NewHub(client, sentry.NewScope()), transport
}

// attachHub returns chi middleware that pins the given Hub on the request
// context so the downstream sentryhttp middleware adopts it instead of cloning
// the global hub.
//
// [Ja] sentryhttp がグローバル Hub を clone するのを避けるため、テスト用 Hub を
// リクエスト context に積むミドルウェアを返す。
func attachHub(hub *sentry.Hub) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := sentry.SetHubOnContext(r.Context(), hub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// buildRouter constructs a router whose middleware chain mirrors the production
// setup in cmd/server/main.go around Sentry (Recoverer → sentryhttp →
// SentryTransaction) plus an attachHub shim used only by tests.
//
// [Ja] 本番の cmd/server/main.go と同じ Sentry 周りのチェーン
// (Recoverer → sentryhttp → SentryTransaction) を組んだルーターを作る。
// テスト用の Hub 差し込み (attachHub) のみ追加で噛ませる。
func buildRouter(hub *sentry.Hub, register func(chi.Router)) *chi.Mux {
	sentryHTTP := sentryhttp.New(sentryhttp.Options{Repanic: true})
	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(attachHub(hub))
	r.Use(sentryHTTP.Handle)
	r.Use(middleware.SentryTransaction)
	register(r)
	return r
}

func findEvents(events []*sentry.Event, eventType string) []*sentry.Event {
	var out []*sentry.Event
	for _, e := range events {
		if eventType == errorEventType && e.Type != "transaction" {
			out = append(out, e)
			continue
		}
		if e.Type == eventType {
			out = append(out, e)
		}
	}
	return out
}

func TestSentryTransaction_PanicEventCarriesRoutePattern(t *testing.T) {
	t.Parallel()

	hub, transport := newTestHub(t)

	router := buildRouter(hub, func(r chi.Router) {
		r.Get("/items/{id}", func(_ http.ResponseWriter, _ *http.Request) {
			panic(errors.New("boom"))
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/items/abc", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// chi's Recoverer must absorb the re-panicked error and return 500.
	// [Ja] chi の Recoverer が再 panic を握り潰し、500 を返す経路を確認する。
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}

	hub.Flush(2 * time.Second)
	events := transport.Events()

	errEvents := findEvents(events, errorEventType)
	if len(errEvents) != 1 {
		t.Fatalf("expected 1 error event, got %d (events=%+v)", len(errEvents), events)
	}
	if got, want := errEvents[0].Transaction, "GET /items/{id}"; got != want {
		t.Errorf("error event Transaction = %q, want %q", got, want)
	}

	txEvents := findEvents(events, "transaction")
	if len(txEvents) != 1 {
		t.Fatalf("expected 1 transaction event, got %d", len(txEvents))
	}
	if got, want := txEvents[0].Transaction, "GET /items/{id}"; got != want {
		t.Errorf("transaction event Transaction = %q, want %q", got, want)
	}
	if got, want := txEvents[0].TransactionInfo, (&sentry.TransactionInfo{Source: sentry.SourceRoute}); got == nil || got.Source != want.Source {
		t.Errorf("transaction event TransactionInfo.Source = %+v, want %+v", got, want)
	}
}

func TestSentryTransaction_CapturedErrorCarriesRoutePattern(t *testing.T) {
	t.Parallel()

	hub, transport := newTestHub(t)

	router := buildRouter(hub, func(r chi.Router) {
		r.Get("/s/{space_identifier}/topics/{topic_number}", func(w http.ResponseWriter, req *http.Request) {
			ctxHub := sentry.GetHubFromContext(req.Context())
			if ctxHub == nil {
				t.Error("hub missing from request context")
				return
			}
			ctxHub.CaptureException(errors.New("topic lookup failed"))
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/s/foo/topics/42", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	hub.Flush(2 * time.Second)
	events := transport.Events()

	errEvents := findEvents(events, errorEventType)
	if len(errEvents) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(errEvents))
	}

	// The handler captures the exception while it is still running, i.e.
	// before the defer in SentryTransaction has updated the span. The route
	// pattern still ends up on the event because SentryTransaction installs
	// an EventProcessor that reads chi.RouteContext().RoutePattern() at
	// capture time -- and chi has already populated that pattern by the time
	// the handler runs.
	//
	// [Ja] ハンドラー実行中の CaptureException は本ミドルウェアの defer より
	// 先に走るが、SentryTransaction が仕込んだ EventProcessor が
	// chi.RouteContext().RoutePattern() をキャプチャ時に読むため Transaction
	// が乗る。chi はハンドラー実行時点で既にルートパターンを確定させている。
	if got, want := errEvents[0].Transaction, "GET /s/{space_identifier}/topics/{topic_number}"; got != want {
		t.Errorf("error event Transaction = %q, want %q", got, want)
	}
}

func TestSentryTransaction_NoChiContext_NoOp(t *testing.T) {
	t.Parallel()

	// Direct invocation without chi guarantees there is no route context to
	// read. The middleware must silently no-op rather than panic, so calls
	// from static-file or non-chi paths remain safe.
	//
	// [Ja] chi を介さず直接呼び出すと RouteContext が無い状態になる。本
	// ミドルウェアはそのまま no-op で通すこと (静的ファイル等で安全に動く)。
	hub, transport := newTestHub(t)

	called := false
	handler := middleware.SentryTransaction(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	req = req.WithContext(sentry.SetHubOnContext(req.Context(), hub))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("downstream handler was not invoked")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	hub.Flush(100 * time.Millisecond)
	if len(transport.Events()) != 0 {
		t.Errorf("did not expect any events, got %d", len(transport.Events()))
	}
}

func TestSentryTransaction_UnmatchedRoute_NoOp(t *testing.T) {
	t.Parallel()

	// When chi cannot match the URL, RoutePattern returns "" and there is no
	// pattern to record. The middleware must still let the downstream 404
	// handler run instead of overwriting Transaction with a misleading value.
	//
	// [Ja] chi がマッチできなかった場合 RoutePattern は "" になる。本ミドル
	// ウェアは何も上書きせず、404 ハンドラーをそのまま走らせる。
	hub, _ := newTestHub(t)

	router := buildRouter(hub, func(r chi.Router) {
		r.Get("/known", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/does/not/exist", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// withUser injects a *model.User into the request context so SentryUserContext
// sees a value as if the auth middleware had already populated it. We bypass
// the real auth middleware in tests because it depends on the session store,
// and these tests focus only on the Sentry plumbing.
//
// [Ja] テストでは認証ミドルウェアの代わりに本ヘルパーでユーザーを context に
// 注入する。実物の認証ミドルウェアはセッションストアに依存しており、本テスト
// の関心 (Sentry スコープへの紐付け) からは外れるためモック扱いする。
func withUser(user *model.User) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.SetUserToContext(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// captureUserViaException makes the handler capture an exception so we can
// inspect the resulting Sentry event's User field. Reading hub.Scope().user
// directly is not possible (the field is unexported); going through a real
// CaptureException is the same path production code takes, so the assertion
// exercises the exact code path that matters.
//
// [Ja] ハンドラー内で意図的に例外を Capture することで、Sentry イベントの
// User フィールドを検証できるようにする。hub.Scope() の user フィールドは
// 非公開のため直接読めないが、本番と同じ CaptureException 経由でユーザー
// 情報が乗ることを確認できる。
func captureUserViaException(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			t.Error("hub missing from request context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		hub.CaptureException(errors.New("user context probe"))
		w.WriteHeader(http.StatusOK)
	}
}

func TestSentryUserContext_SetsUser_WithAtname(t *testing.T) {
	t.Parallel()

	hub, transport := newTestHub(t)

	user := &model.User{ID: model.UserID("user-with-atname"), Atname: "alice"}

	router := chi.NewRouter()
	router.Use(attachHub(hub))
	router.Use(withUser(user))
	router.Use(middleware.SentryUserContext)
	router.Get("/probe", captureUserViaException(t))

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	hub.Flush(2 * time.Second)
	events := findEvents(transport.Events(), errorEventType)
	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}
	got := events[0].User
	if got.ID != "user-with-atname" {
		t.Errorf("User.ID = %q, want %q", got.ID, "user-with-atname")
	}
	if got.Username != "alice" {
		t.Errorf("User.Username = %q, want %q", got.Username, "alice")
	}
}

func TestSentryUserContext_SetsUserIDOnly_WhenAtnameEmpty(t *testing.T) {
	t.Parallel()

	hub, transport := newTestHub(t)

	// Atname can be empty for users imported before the atname column was
	// introduced. They still have a stable ID, so the event must carry that ID
	// but leave Username unset (rather than sending an empty string).
	//
	// [Ja] atname カラム導入前の移行データなどでは Atname が空になり得る。
	// ID は安定しているため ID は埋め、Username は空文字列ではなく未設定の
	// まま送信されることを確認する。
	user := &model.User{ID: model.UserID("legacy-user"), Atname: ""}

	router := chi.NewRouter()
	router.Use(attachHub(hub))
	router.Use(withUser(user))
	router.Use(middleware.SentryUserContext)
	router.Get("/probe", captureUserViaException(t))

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	hub.Flush(2 * time.Second)
	events := findEvents(transport.Events(), errorEventType)
	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}
	got := events[0].User
	if got.ID != "legacy-user" {
		t.Errorf("User.ID = %q, want %q", got.ID, "legacy-user")
	}
	if got.Username != "" {
		t.Errorf("User.Username = %q, want empty", got.Username)
	}
}

func TestSentryUserContext_NoOp_WhenUnauthenticated(t *testing.T) {
	t.Parallel()

	hub, transport := newTestHub(t)

	// No withUser() in the chain, so UserFromContext returns nil. The middleware
	// must skip SetUser entirely; otherwise anonymous traffic would inherit
	// whatever the scope was last populated with.
	//
	// [Ja] withUser() を噛ませないため UserFromContext は nil を返す。
	// 本ミドルウェアは SetUser を呼ばずに通過しなければならない。さもないと
	// 匿名トラフィックが直前のスコープのユーザー情報を引き継ぐリスクがある。
	router := chi.NewRouter()
	router.Use(attachHub(hub))
	router.Use(middleware.SentryUserContext)
	router.Get("/probe", captureUserViaException(t))

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	hub.Flush(2 * time.Second)
	events := findEvents(transport.Events(), errorEventType)
	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}
	if !events[0].User.IsEmpty() {
		t.Errorf("expected empty user, got %+v", events[0].User)
	}
}

func TestSentryUserContext_NoOp_WhenHubMissing(t *testing.T) {
	t.Parallel()

	// Direct invocation without attachHub leaves the context without a Sentry
	// Hub (this is what happens for paths served outside the main chain, such
	// as static files). The middleware must not panic and must let the
	// downstream handler run normally.
	//
	// [Ja] attachHub を噛ませず直接呼ぶと context に Hub が無い状態になる
	// (静的ファイル経路などで起こり得る)。本ミドルウェアは panic せず
	// 下流ハンドラーをそのまま走らせなければならない。
	user := &model.User{ID: model.UserID("user-without-hub"), Atname: "bob"}

	called := false
	handler := middleware.SentryUserContext(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	req = req.WithContext(middleware.SetUserToContext(req.Context(), user))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("downstream handler was not invoked")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSentryUserContext_PropagatesUserAcrossHandlers(t *testing.T) {
	t.Parallel()

	// The middleware runs once at request entry but the user must remain on the
	// Hub scope for any exception captured deeper in the chain -- e.g. by a
	// downstream middleware or a nested handler. We verify this by chaining a
	// second middleware after SentryUserContext that captures the exception
	// itself; if the user information did not propagate, the captured event
	// would arrive without User set.
	//
	// [Ja] 本ミドルウェアはリクエスト入口で一度だけ走るが、その後の下流
	// ミドルウェアや子ハンドラーがキャプチャするイベントにもユーザー情報が
	// 乗っている必要がある。SentryUserContext の後にもう 1 つ
	// ミドルウェアを挟み、そこで CaptureException を呼ぶことで、スコープに
	// 紐付いたユーザーが正しく後段まで伝搬していることを確認する。
	hub, transport := newTestHub(t)

	user := &model.User{ID: model.UserID("propagated-user"), Atname: "carol"}

	captureDeeper := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if h := sentry.GetHubFromContext(r.Context()); h != nil {
				h.CaptureException(errors.New("captured from a later middleware"))
			}
			next.ServeHTTP(w, r)
		})
	}

	router := chi.NewRouter()
	router.Use(attachHub(hub))
	router.Use(withUser(user))
	router.Use(middleware.SentryUserContext)
	router.Use(captureDeeper)
	router.Get("/probe", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	hub.Flush(2 * time.Second)
	events := findEvents(transport.Events(), errorEventType)
	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}
	got := events[0].User
	if got.ID != "propagated-user" {
		t.Errorf("User.ID = %q, want %q", got.ID, "propagated-user")
	}
	if got.Username != "carol" {
		t.Errorf("User.Username = %q, want %q", got.Username, "carol")
	}
}
