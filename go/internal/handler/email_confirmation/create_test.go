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

// mockTurnstileVerifier はテスト用のTurnstile検証モック
type mockTurnstileVerifier struct {
	valid bool
	err   error
}

func (m *mockTurnstileVerifier) Verify(_ context.Context, _ string) (bool, error) {
	return m.valid, m.err
}

// mockJobInserter はテスト用のモック inserter
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/email_confirmation/edit" {
		t.Errorf("wrong redirect location: got %v want /email_confirmation/edit", location)
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
		t.Error("email_confirmation_id cookie not set")
	}

	if !setup.mockInserter.called {
		t.Error("Insert was not called")
	}
	emailArgs, ok := setup.mockInserter.args.(dispatcher.SendEmailConfirmationArgs)
	if !ok {
		t.Fatalf("args の型が SendEmailConfirmationArgs ではありません: %T", setup.mockInserter.args)
	}
	if emailArgs.Email != "newuser@example.com" {
		t.Errorf("enqueued email = %s, want newuser@example.com", emailArgs.Email)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == session.EmailConfirmationCookieName {
			t.Error("email_confirmation_id cookie should not be set for Turnstile failure")
		}
	}

	if setup.mockInserter.called {
		t.Error("Insert should not be called")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `action="/email_confirmation"`) {
		t.Error("sign up form not found in response")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "既に登録されています") {
		t.Error("email already registered error message not found in response")
	}

	if setup.mockInserter.called {
		t.Error("Insert should not be called for existing email")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	if !setup.mockInserter.called {
		t.Error("Insert was not called")
	}
}

func TestCreate_RateLimitExceeded_IP(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, true)

	// 同じIPから5回リクエスト（制限内）
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
			t.Errorf("request %d: wrong status code: got %v want %v", i+1, rr.Code, http.StatusFound)
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
		t.Errorf("6th request: wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("rate limit exceeded message not found in response")
	}
}

// TestCreate_RateLimit_PrioritizesCFConnectingIP locks in that the IP rate-limit
// key is derived from internal/clientip (CF-Connecting-IP first). Requests that
// share the same CF-Connecting-IP fall into one bucket even when X-Forwarded-For
// differs, so the limit triggers on the 6th request; if the handler keyed on
// X-Forwarded-For instead, each request would land in a separate bucket and never
// be limited.
//
// [Ja] IP レート制限キーが internal/clientip (CF-Connecting-IP 優先) から導出される
// ことを固定する。X-Forwarded-For が異なっても CF-Connecting-IP が同じなら同一バケットに
// 入り、6 回目で制限が発火する。もしハンドラーが X-Forwarded-For をキーにしていたら、
// 各リクエストは別バケットになり制限されない。
func TestCreate_RateLimit_PrioritizesCFConnectingIP(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, true)

	const cfConnectingIP = "203.0.113.10"

	// Stay within the limits: 5 requests (IP cap is 5/hour), each with a distinct
	// email to avoid the per-email cap (3/hour) and a varying X-Forwarded-For so
	// only the shared CF-Connecting-IP can be the rate-limit key.
	// [Ja] 制限内に収める: IP 上限 (5/時間) ぶんの 5 回。メール上限 (3/時間) を避けるため
	// 各リクエストは別メールにし、X-Forwarded-For を毎回変えることで、共通の
	// CF-Connecting-IP だけがレート制限キーになり得る状況を作る。
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
			t.Errorf("request %d: wrong status code: got %v want %v", i+1, rr.Code, http.StatusFound)
		}
	}

	// The 6th request shares the CF-Connecting-IP (with yet another X-Forwarded-For),
	// so it is rejected by the IP rate limit.
	// [Ja] 6 回目は CF-Connecting-IP が同じ (X-Forwarded-For はさらに別) なので IP 制限で拒否される。
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
		t.Errorf("6th request: wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("rate limit exceeded message not found in response")
	}
}

func TestCreate_RateLimitExceeded_Email(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	setup := newTestHandlerForCreate(t, queries, true)

	// 同じメールアドレスで3回リクエスト（制限内、異なるIPから）
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
			t.Errorf("request %d: wrong status code: got %v want %v", i+1, rr.Code, http.StatusFound)
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
		t.Errorf("4th request: wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("rate limit exceeded message not found in response")
	}
}
