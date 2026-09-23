package email_confirmation_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/handler/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// mockTurnstileVerifierはテスト用のTurnstile検証モック
type mockTurnstileVerifier struct {
	valid bool
	err   error
}

func (m *mockTurnstileVerifier) Verify(_ context.Context, _ string) (bool, error) {
	return m.valid, m.err
}

// mockJobInserterはテスト用のモックinserter
type mockJobInserter struct {
	called bool
	args   river.JobArgs
}

func (m *mockJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	m.called = true
	m.args = args
	return &rivertype.JobInsertResult{}, nil
}

type testHandlerSetup struct {
	handler      *email_confirmation.Handler
	mockInserter *mockJobInserter
}

func newTestHandlerForCreate(t *testing.T, queries *query.Queries, turnstileValid bool) *testHandlerSetup {
	t.Helper()

	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "",
		TurnstileSecretKey: "",
	}

	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	mock := &mockJobInserter{}
	d := dispatcher.NewDispatcher(mock)
	emailConfirmationCreateValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	emailConfirmationUpdateValidator := validator.NewEmailConfirmationUpdateValidator(emailConfirmationRepo)
	createEmailConfirmationUC := usecase.NewCreateEmailConfirmationUsecase(cfg, emailConfirmationRepo, d, emailConfirmationCreateValidator)
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo, emailConfirmationUpdateValidator)

	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	mockTurnstile := &mockTurnstileVerifier{valid: turnstileValid, err: nil}

	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		createEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)

	return &testHandlerSetup{handler: handler, mockInserter: mock}
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, true)

	form := url.Values{}
	form.Set("email", "newuser@example.com")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/email_confirmation/edit" {
		t.Errorf("リダイレクト先 = %v、期待値 = /email_confirmation/edit", location)
	}

	cookies := rr.Result().Cookies()
	var emailConfirmationCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.EmailConfirmationCookieName {
			emailConfirmationCookie = c
			break
		}
	}
	if emailConfirmationCookie == nil {
		t.Error("email_confirmation_idのCookieがセットされていない")
	}

	if !setup.mockInserter.called {
		t.Error("Insertが呼ばれていない")
	}
	emailArgs, ok := setup.mockInserter.args.(dispatcher.SendEmailConfirmationArgs)
	if !ok {
		t.Fatalf("argsの型がSendEmailConfirmationArgsではありません: %T", setup.mockInserter.args)
	}
	if emailArgs.Email != "newuser@example.com" {
		t.Errorf("キューに入ったメールアドレス = %s、期待値 = newuser@example.com", emailArgs.Email)
	}
}

func TestCreate_TurnstileFailure(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, false)

	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == session.EmailConfirmationCookieName {
			t.Error("Turnstileの失敗時にemail_confirmation_idのCookieがセットされている")
		}
	}

	if setup.mockInserter.called {
		t.Error("Insertが呼ばれている")
	}
}

func TestCreate_InvalidEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, true)

	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `action="/email_confirmation"`) {
		t.Error("レスポンスにサインアップのフォームが見つからない")
	}
}

func TestCreate_EmptyEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, true)

	form := url.Values{}
	form.Set("email", "")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestCreate_EmailAlreadyRegistered(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("existinguser1@example.com").
		WithAtname("existinguser1").
		Build()

	setup := newTestHandlerForCreate(t, queries, true)

	form := url.Values{}
	form.Set("email", "existinguser1@example.com")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "既に登録されています") {
		t.Error("レスポンスにメールアドレス登録済みのエラーメッセージが見つからない")
	}

	if setup.mockInserter.called {
		t.Error("既存のメールアドレスでInsertが呼ばれている")
	}
}

func TestCreate_PasswordResetEvent_AllowsExistingEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("resetuser@example.com").
		WithAtname("resetuser").
		Build()

	setup := newTestHandlerForCreate(t, queries, true)

	form := url.Values{}
	form.Set("email", "resetuser@example.com")
	form.Set("event", "password_reset")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	if !setup.mockInserter.called {
		t.Error("Insertが呼ばれていない")
	}
}

func TestCreate_RateLimitExceeded_IP(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, true)

	// 同じIPから5回リクエスト (制限内)
	for i := 0; i < 5; i++ {
		form := url.Values{}
		form.Set("email", "user"+string(rune('a'+i))+"@example.com")
		form.Set("event", "signup")
		form.Set("csrf_token", "test-csrf-token")
		form.Set("cf-turnstile-response", "test-token")

		req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Forwarded-For", "192.168.1.100")

		ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
		ctx = i18n.SetLocale(ctx, "ja")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		setup.handler.Create(rr, req)

		if rr.Code != http.StatusFound {
			t.Errorf("リクエスト%d: ステータスコード = %v、期待値 = %v", i+1, rr.Code, http.StatusFound)
		}
	}

	// 6回目のリクエストはRate Limitで拒否される
	form := url.Values{}
	form.Set("email", "userf@example.com")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("6回目のリクエスト: ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("レスポンスにレート制限超過のメッセージが見つからない")
	}
}

// IPレート制限キーがinternal/clientip (CF-Connecting-IP優先) から導出される
// ことを固定する。X-Forwarded-Forが異なってもCF-Connecting-IPが同じなら同一バケットに
// 入り、6回目で制限が発火する。もしハンドラーがX-Forwarded-Forをキーにしていたら、
// 各リクエストは別バケットになり制限されない。
func TestCreate_RateLimit_PrioritizesCFConnectingIP(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, true)

	const cfConnectingIP = "203.0.113.10"

	// 制限内に収める: IP上限 (5/時間) ぶんの5回。メール上限 (3/時間) を避けるため
	// 各リクエストは別メールにし、X-Forwarded-Forを毎回変えることで、共通の
	// CF-Connecting-IPだけがレート制限キーになり得る状況を作る。
	for i := 0; i < 5; i++ {
		form := url.Values{}
		form.Set("email", fmt.Sprintf("cfip%d@example.com", i))
		form.Set("event", "signup")
		form.Set("csrf_token", "test-csrf-token")
		form.Set("cf-turnstile-response", "test-token")

		req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("CF-Connecting-IP", cfConnectingIP)
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i+1))

		ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
		ctx = i18n.SetLocale(ctx, "ja")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		setup.handler.Create(rr, req)

		if rr.Code != http.StatusFound {
			t.Errorf("リクエスト%d: ステータスコード = %v、期待値 = %v", i+1, rr.Code, http.StatusFound)
		}
	}

	// 6回目はCF-Connecting-IPが同じ (X-Forwarded-Forはさらに別) なのでIP制限で拒否される。
	form := url.Values{}
	form.Set("email", "cfip5@example.com")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("CF-Connecting-IP", cfConnectingIP)
	req.Header.Set("X-Forwarded-For", "198.51.100.200")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("6回目のリクエスト: ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("レスポンスにレート制限超過のメッセージが見つからない")
	}
}

func TestCreate_RateLimitExceeded_Email(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, true)

	// 同じメールアドレスで3回リクエスト (制限内、異なるIPから)
	for i := 0; i < 3; i++ {
		form := url.Values{}
		form.Set("email", "sameuser@example.com")
		form.Set("event", "signup")
		form.Set("csrf_token", "test-csrf-token")
		form.Set("cf-turnstile-response", "test-token")

		req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Forwarded-For", "192.168.1."+string(rune('1'+i)))

		ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
		ctx = i18n.SetLocale(ctx, "ja")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		setup.handler.Create(rr, req)

		if rr.Code != http.StatusFound {
			t.Errorf("リクエスト%d: ステータスコード = %v、期待値 = %v", i+1, rr.Code, http.StatusFound)
		}
	}

	// 4回目のリクエストはRate Limitで拒否される
	form := url.Values{}
	form.Set("email", "sameuser@example.com")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", "192.168.1.99")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("4回目のリクエスト: ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("レスポンスにレート制限超過のメッセージが見つからない")
	}
}
