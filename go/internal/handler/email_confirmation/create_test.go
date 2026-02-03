package email_confirmation_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/worker"
)

// mockTurnstileVerifier はテスト用のTurnstile検証モック
type mockTurnstileVerifier struct {
	valid bool
	err   error
}

func (m *mockTurnstileVerifier) Verify(_ context.Context, _ string) (bool, error) {
	return m.valid, m.err
}

// mockInserter はテスト用のモック inserter
type mockInserter struct {
	called bool
	args   river.JobArgs
}

func (m *mockInserter) Insert(_ context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	m.called = true
	m.args = args
	return &rivertype.JobInsertResult{}, nil
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
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

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserter{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ユースケースを初期化
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("email", "newuser@example.com")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/email_confirmation/edit" {
		t.Errorf("wrong redirect location: got %v want /email_confirmation/edit", location)
	}

	// email_confirmation_id Cookieが設定されているか確認
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

	// エンキューが呼ばれたことを確認
	if !inserter.called {
		t.Error("Insert was not called")
	}
	emailArgs, ok := inserter.args.(worker.SendEmailArgs)
	if !ok {
		t.Fatalf("args の型が SendEmailArgs ではありません: %T", inserter.args)
	}
	if emailArgs.To != "newuser@example.com" {
		t.Errorf("enqueued email = %s, want newuser@example.com", emailArgs.To)
	}
}

func TestCreate_TurnstileFailure(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
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

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserter{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に失敗）
	mockTurnstile := &mockTurnstileVerifier{valid: false, err: nil}

	// ユースケースを初期化
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// email_confirmation_id Cookieが設定されていないことを確認
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == session.EmailConfirmationCookieName {
			t.Error("email_confirmation_id cookie should not be set for Turnstile failure")
		}
	}

	// エンキューが呼ばれていないことを確認
	if inserter.called {
		t.Error("Insert should not be called")
	}
}

func TestCreate_InvalidEmail(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
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

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserter{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ユースケースを初期化
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)

	// 無効なメールアドレスでリクエスト
	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// サインアップフォームが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `action="/email_confirmation"`) {
		t.Error("sign up form not found in response")
	}
}

func TestCreate_EmptyEmail(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
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

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserter{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ユースケースを初期化
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)

	// 空のメールアドレスでリクエスト
	form := url.Values{}
	form.Set("email", "")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestCreate_EmailAlreadyRegistered(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 既存のユーザーを作成
	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("existinguser1@example.com").
		WithAtname("existinguser1").
		Build()

	// 設定を作成
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

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserter{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ユースケースを初期化
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)

	// 既に登録済みのメールアドレスでリクエスト
	form := url.Values{}
	form.Set("email", "existinguser1@example.com")
	form.Set("event", "signup")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "既に登録されています") {
		t.Error("email already registered error message not found in response")
	}

	// エンキューが呼ばれていないことを確認
	if inserter.called {
		t.Error("Insert should not be called for existing email")
	}
}

func TestCreate_PasswordResetEvent_AllowsExistingEmail(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 既存のユーザーを作成
	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("resetuser@example.com").
		WithAtname("resetuser").
		Build()

	// 設定を作成
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

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserter{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ユースケースを初期化
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)

	// password_reset イベントでは既存のメールアドレスでもOK
	form := url.Values{}
	form.Set("email", "resetuser@example.com")
	form.Set("event", "password_reset")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "test-token")

	req := httptest.NewRequest(http.MethodPost, "/email_confirmation", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// エンキューが呼ばれたことを確認
	if !inserter.called {
		t.Error("Insert was not called")
	}
}

func TestCreate_RateLimitExceeded_IP(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
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

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserter{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ユースケースを初期化
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)

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
		handler.Create(rr, req)

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
	handler.Create(rr, req)

	// Rate Limit超過時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("6th request: wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("rate limit exceeded message not found in response")
	}
}

func TestCreate_RateLimitExceeded_Email(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTestDB(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
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

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	inserter := &mockInserter{}
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, inserter)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ユースケースを初期化
	markEmailAsConfirmedUC := usecase.NewMarkEmailAsConfirmedUsecase(emailConfirmationRepo)

	// Rate Limiterを初期化
	limiter := ratelimit.NewLimiter(queries)

	// ハンドラーを初期化
	handler := email_confirmation.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		emailConfirmationRepo,
		sendEmailConfirmationUC,
		markEmailAsConfirmedUC,
		mockTurnstile,
		limiter,
	)

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
		handler.Create(rr, req)

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
	handler.Create(rr, req)

	// Rate Limit超過時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("4th request: wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") {
		t.Error("rate limit exceeded message not found in response")
	}
}
