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

// Sentryクライアントが本来ネットワーク送信するイベントをすべて収集する
// テスト用Transport。transaction.FinishとrecoverWithSentryが別ゴルーチンから
// SendEventを呼ぶ可能性があるため排他制御で守る。
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

// テストごとに独立したHub + captureTransportを作る。グローバルHubには
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
		t.Fatalf("sentry.NewClientのエラー = %v", err)
	}
	return sentry.NewHub(client, sentry.NewScope()), transport
}

// sentryhttpがグローバルHubをcloneするのを避けるため、テスト用Hubを
// リクエストcontextに積むミドルウェアを返す。
func attachHub(hub *sentry.Hub) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := sentry.SetHubOnContext(r.Context(), hub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// 本番のcmd/wikino/serve.goと同じSentry周りのチェーン
// (Recoverer → sentryhttp → SentryTransaction) を組んだルーターを作る。
// テスト用のHub差し込み (attachHub) のみ追加で噛ませる。
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

	// chiのRecovererが再panicを握り潰し、500を返す経路を確認する。
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusInternalServerError)
	}

	hub.Flush(2 * time.Second)
	events := transport.Events()

	errEvents := findEvents(events, errorEventType)
	if len(errEvents) != 1 {
		t.Fatalf("エラーイベントの件数 = %d、期待値 = 1 (events = %+v)", len(errEvents), events)
	}
	if got, want := errEvents[0].Transaction, "GET /items/{id}"; got != want {
		t.Errorf("エラーイベントのTransaction = %q、期待値 = %q", got, want)
	}

	txEvents := findEvents(events, "transaction")
	if len(txEvents) != 1 {
		t.Fatalf("トランザクションイベントの件数 = %d、期待値 = 1", len(txEvents))
	}
	if got, want := txEvents[0].Transaction, "GET /items/{id}"; got != want {
		t.Errorf("トランザクションイベントのTransaction = %q、期待値 = %q", got, want)
	}
	if got, want := txEvents[0].TransactionInfo, (&sentry.TransactionInfo{Source: sentry.SourceRoute}); got == nil || got.Source != want.Source {
		t.Errorf("トランザクションイベントのTransactionInfo.Source = %+v、期待値 = %+v", got, want)
	}
}

func TestSentryTransaction_CapturedErrorCarriesRoutePattern(t *testing.T) {
	t.Parallel()

	hub, transport := newTestHub(t)

	router := buildRouter(hub, func(r chi.Router) {
		r.Get("/s/{space_identifier}/topics/{topic_number}", func(w http.ResponseWriter, req *http.Request) {
			ctxHub := sentry.GetHubFromContext(req.Context())
			if ctxHub == nil {
				t.Error("リクエストのコンテキストにhubが無い")
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
		t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	hub.Flush(2 * time.Second)
	events := transport.Events()

	errEvents := findEvents(events, errorEventType)
	if len(errEvents) != 1 {
		t.Fatalf("エラーイベントの件数 = %d、期待値 = 1", len(errEvents))
	}

	// ハンドラー実行中のCaptureExceptionは本ミドルウェアのdeferより
	// 先に走るが、SentryTransactionが仕込んだEventProcessorが
	// chi.RouteContext().RoutePattern() をキャプチャ時に読むためTransaction
	// が乗る。chiはハンドラー実行時点で既にルートパターンを確定させている。
	if got, want := errEvents[0].Transaction, "GET /s/{space_identifier}/topics/{topic_number}"; got != want {
		t.Errorf("エラーイベントのTransaction = %q、期待値 = %q", got, want)
	}
}

func TestSentryTransaction_NoChiContext_NoOp(t *testing.T) {
	t.Parallel()

	// chiを介さず直接呼び出すとRouteContextが無い状態になる。本
	// ミドルウェアはそのままno-opで通すこと (静的ファイル等で安全に動く)。
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
		t.Fatal("後続のハンドラーが呼ばれていない")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	hub.Flush(100 * time.Millisecond)
	if len(transport.Events()) != 0 {
		t.Errorf("イベントの件数 = %d、期待値 = 0", len(transport.Events()))
	}
}

func TestSentryTransaction_UnmatchedRoute_NoOp(t *testing.T) {
	t.Parallel()

	// chiがマッチできなかった場合RoutePatternは "" になる。本ミドル
	// ウェアは何も上書きせず、404ハンドラーをそのまま走らせる。
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
		t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
}

// テストでは認証ミドルウェアの代わりに本ヘルパーでユーザーをcontextに
// 注入する。実物の認証ミドルウェアはセッションストアに依存しており、本テスト
// の関心 (Sentryスコープへの紐付け) からは外れるためモック扱いする。
func withUser(user *model.User) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.SetUserToContext(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ハンドラー内で意図的に例外をCaptureすることで、Sentryイベントの
// Userフィールドを検証できるようにする。hub.Scope() のuserフィールドは
// 非公開のため直接読めないが、本番と同じCaptureException経由でユーザー
// 情報が乗ることを確認できる。
func captureUserViaException(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			t.Error("リクエストのコンテキストにhubが無い")
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
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	hub.Flush(2 * time.Second)
	events := findEvents(transport.Events(), errorEventType)
	if len(events) != 1 {
		t.Fatalf("エラーイベントの件数 = %d、期待値 = 1", len(events))
	}
	got := events[0].User
	if got.ID != "user-with-atname" {
		t.Errorf("User.ID = %q、期待値 = %q", got.ID, "user-with-atname")
	}
	if got.Username != "alice" {
		t.Errorf("User.Username = %q、期待値 = %q", got.Username, "alice")
	}
}

func TestSentryUserContext_SetsUserIDOnly_WhenAtnameEmpty(t *testing.T) {
	t.Parallel()

	hub, transport := newTestHub(t)

	// atnameカラム導入前の移行データなどではAtnameが空になり得る。
	// IDは安定しているためIDは埋め、Usernameは空文字列ではなく未設定の
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
		t.Fatalf("エラーイベントの件数 = %d、期待値 = 1", len(events))
	}
	got := events[0].User
	if got.ID != "legacy-user" {
		t.Errorf("User.ID = %q、期待値 = %q", got.ID, "legacy-user")
	}
	if got.Username != "" {
		t.Errorf("User.Username = %q、期待値 = 空", got.Username)
	}
}

func TestSentryUserContext_NoOp_WhenUnauthenticated(t *testing.T) {
	t.Parallel()

	hub, transport := newTestHub(t)

	// withUser() を噛ませないためUserFromContextはnilを返す。
	// 本ミドルウェアはSetUserを呼ばずに通過しなければならない。さもないと
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
		t.Fatalf("エラーイベントの件数 = %d、期待値 = 1", len(events))
	}
	if !events[0].User.IsEmpty() {
		t.Errorf("ユーザー = %+v、期待値 = 空", events[0].User)
	}
}

func TestSentryUserContext_NoOp_WhenHubMissing(t *testing.T) {
	t.Parallel()

	// attachHubを噛ませず直接呼ぶとcontextにHubが無い状態になる
	// (静的ファイル経路などで起こり得る)。本ミドルウェアはpanicせず
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
		t.Fatal("後続のハンドラーが呼ばれていない")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
}

func TestSentryUserContext_PropagatesUserAcrossHandlers(t *testing.T) {
	t.Parallel()

	// 本ミドルウェアはリクエスト入口で一度だけ走るが、その後の下流
	// ミドルウェアや子ハンドラーがキャプチャするイベントにもユーザー情報が
	// 乗っている必要がある。SentryUserContextの後にもう1つ
	// ミドルウェアを挟み、そこでCaptureExceptionを呼ぶことで、スコープに
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
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	hub.Flush(2 * time.Second)
	events := findEvents(transport.Events(), errorEventType)
	if len(events) != 1 {
		t.Fatalf("エラーイベントの件数 = %d、期待値 = 1", len(events))
	}
	got := events[0].User
	if got.ID != "propagated-user" {
		t.Errorf("User.ID = %q、期待値 = %q", got.ID, "propagated-user")
	}
	if got.Username != "carol" {
		t.Errorf("User.Username = %q、期待値 = %q", got.Username, "carol")
	}
}
