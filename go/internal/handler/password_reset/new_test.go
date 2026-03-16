package password_reset_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/password_reset"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestNew_Success(t *testing.T) {
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

	// バリデーターを初期化
	passwordResetCreateValidator := validator.NewPasswordResetCreateValidator()

	// ハンドラーを初期化
	handler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		nil, // limiter
		nil, // turnstileVerifier
		nil, // createTokenUsecase
		passwordResetCreateValidator,
	)

	req := httptest.NewRequest(http.MethodGet, "/password/reset", nil)

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// フォームが表示されているか確認
	body := rr.Body.String()
	if !strings.Contains(body, `action="/password/reset"`) {
		t.Error("password reset form not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "csrf_token") {
		t.Error("CSRF token not found in form")
	}

	// Turnstileウィジェットが含まれているか確認
	if !strings.Contains(body, "cf-turnstile") {
		t.Error("Turnstile widget not found in form")
	}
}

func TestNew_I18n_Japanese(t *testing.T) {
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

	// バリデーターを初期化
	passwordResetCreateValidator := validator.NewPasswordResetCreateValidator()

	// ハンドラーを初期化
	handler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		nil, // limiter
		nil, // turnstileVerifier
		nil, // createTokenUsecase
		passwordResetCreateValidator,
	)

	req := httptest.NewRequest(http.MethodGet, "/password/reset", nil)

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "ja")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// 日本語のテキストが含まれているか確認
	body := rr.Body.String()
	expectedTexts := []string{
		"パスワードリセット",
		"メールアドレス",
		"ログインに戻る",
	}

	for _, expected := range expectedTexts {
		if !strings.Contains(body, expected) {
			t.Errorf("expected text not found: %s", expected)
		}
	}
}

func TestNew_I18n_English(t *testing.T) {
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

	// バリデーターを初期化
	passwordResetCreateValidator := validator.NewPasswordResetCreateValidator()

	// ハンドラーを初期化
	handler := password_reset.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		nil, // limiter
		nil, // turnstileVerifier
		nil, // createTokenUsecase
		passwordResetCreateValidator,
	)

	req := httptest.NewRequest(http.MethodGet, "/password/reset", nil)

	// CSRFトークンとロケールをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, "en")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// 英語のテキストが含まれているか確認
	body := rr.Body.String()
	expectedTexts := []string{
		"Password Reset",
		"Email",
		"Back to sign in",
	}

	for _, expected := range expectedTexts {
		if !strings.Contains(body, expected) {
			t.Errorf("expected text not found: %s", expected)
		}
	}
}
