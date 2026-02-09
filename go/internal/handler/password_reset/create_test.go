package password_reset_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/password_reset"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// mockTurnstileVerifier はテスト用のTurnstile検証モック
type mockTurnstileVerifier struct {
	valid bool
	err   error
}

func (m *mockTurnstileVerifier) Verify(ctx context.Context, token string) (bool, error) {
	return m.valid, m.err
}

func TestCreate_TurnstileVerification_Success(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テスト用パスワードをハッシュ化
	password := "testpassword123"
	passwordDigest, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}

	// テストユーザーを作成
	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithAtname("testuser").
		BuildWithPassword(passwordDigest)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "test-secret-key",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ユースケースを初期化
	createTokenUsecase := usecase.NewCreatePasswordResetTokenUsecase(cfg, db, passwordResetTokenRepo, nil)

	// ハンドラーを初期化
	handler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		nil, // limiter（テストでは不要）
		mockTurnstile,
		createTokenUsecase,
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "valid-token")

	req := httptest.NewRequest(http.MethodPost, "/password/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// 成功時はメール送信完了ページを表示
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// メール送信完了ページが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "メールを送信しました") && !strings.Contains(body, "Check your email") {
		t.Error("email sent page not displayed")
	}
}

func TestCreate_TurnstileVerification_Failed(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "test-secret-key",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に失敗）
	mockTurnstile := &mockTurnstileVerifier{valid: false, err: nil}

	// ハンドラーを初期化
	handler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		nil, // limiter
		mockTurnstile,
		nil, // createTokenUsecase（検証失敗のため使われない）
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("email", "test@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "invalid-token")

	req := httptest.NewRequest(http.MethodPost, "/password/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestCreate_ValidationError(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "test-secret-key",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ハンドラーを初期化
	handler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		nil, // limiter
		mockTurnstile,
		nil, // createTokenUsecase
	)

	// 無効なメールアドレスでリクエスト
	form := url.Values{}
	form.Set("email", "invalid-email")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "valid-token")

	req := httptest.NewRequest(http.MethodPost, "/password/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `action="/password/reset"`) {
		t.Error("password reset form not found in response")
	}
}

func TestCreate_RateLimiting_IP(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "test-secret-key",
	}

	// Rate Limiterを初期化（PostgreSQLベース）
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ハンドラーを初期化
	handler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		limiter,
		mockTurnstile,
		nil, // createTokenUsecase（ユーザーが存在しないため使われない）
	)

	// 5回まではOK（IPアドレス単位: 5回/時間）
	for i := 0; i < 5; i++ {
		form := url.Values{}
		form.Set("email", "test"+string(rune('0'+i))+"@example.com")
		form.Set("csrf_token", "test-csrf-token")
		form.Set("cf-turnstile-response", "valid-token")

		req := httptest.NewRequest(http.MethodPost, "/password/reset", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "192.168.1.1:12345"

		ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.Create(rr, req)

		// Rate Limitingエラー以外は成功とみなす
		if rr.Code == http.StatusUnprocessableEntity {
			body := rr.Body.String()
			if strings.Contains(body, "リクエストが多すぎます") || strings.Contains(body, "Too many requests") {
				t.Errorf("attempt %d should not be rate limited yet", i+1)
			}
		}
	}

	// 6回目はRate Limiting
	form := url.Values{}
	form.Set("email", "test6@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "valid-token")

	req := httptest.NewRequest(http.MethodPost, "/password/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "192.168.1.1:12345"

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// Rate Limitingエラー
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("6th attempt should be rate limited, got status %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") && !strings.Contains(body, "Too many requests") {
		t.Error("rate limit error message not found")
	}
}

func TestCreate_RateLimiting_Email(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "test-secret-key",
	}

	// Rate Limiterを初期化（PostgreSQLベース）
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	limiter := ratelimit.NewLimiter(rateLimitRepo)

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ハンドラーを初期化
	handler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		limiter,
		mockTurnstile,
		nil, // createTokenUsecase
	)

	// 3回まではOK（メールアドレス単位: 3回/時間）
	for i := 0; i < 3; i++ {
		form := url.Values{}
		form.Set("email", "ratelimit@example.com")
		form.Set("csrf_token", "test-csrf-token")
		form.Set("cf-turnstile-response", "valid-token")

		req := httptest.NewRequest(http.MethodPost, "/password/reset", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// 異なるIPアドレスから
		req.RemoteAddr = "192.168.1." + string(rune('1'+i)) + ":12345"

		ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.Create(rr, req)

		// Rate Limitingエラー以外は成功とみなす
		if rr.Code == http.StatusUnprocessableEntity {
			body := rr.Body.String()
			if strings.Contains(body, "リクエストが多すぎます") || strings.Contains(body, "Too many requests") {
				t.Errorf("attempt %d should not be rate limited yet", i+1)
			}
		}
	}

	// 4回目はRate Limiting
	form := url.Values{}
	form.Set("email", "ratelimit@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "valid-token")

	req := httptest.NewRequest(http.MethodPost, "/password/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "192.168.1.100:12345"

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// Rate Limitingエラー
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("4th attempt should be rate limited by email, got status %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "リクエストが多すぎます") && !strings.Contains(body, "Too many requests") {
		t.Error("rate limit error message not found")
	}
}

func TestCreate_UserNotExists_ShowsSuccessPage(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:                "test",
		Port:               "8080",
		Domain:             "localhost",
		CookieDomain:       "",
		SessionSecure:      false,
		SessionHTTPOnly:    true,
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "test-secret-key",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化（常に成功）
	mockTurnstile := &mockTurnstileVerifier{valid: true, err: nil}

	// ハンドラーを初期化
	handler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		nil, // limiter
		mockTurnstile,
		nil, // createTokenUsecase
	)

	// 存在しないユーザーのメールアドレスでリクエスト
	form := url.Values{}
	form.Set("email", "nonexistent@example.com")
	form.Set("csrf_token", "test-csrf-token")
	form.Set("cf-turnstile-response", "valid-token")

	req := httptest.NewRequest(http.MethodPost, "/password/reset", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// セキュリティ上、ユーザーが存在しなくても成功ページを表示
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// メール送信完了ページが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "メールを送信しました") && !strings.Contains(body, "Check your email") {
		t.Error("email sent page not displayed for non-existent user")
	}
}
