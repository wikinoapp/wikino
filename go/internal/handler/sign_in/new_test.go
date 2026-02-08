package sign_in_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// mockTurnstileVerifierForNew はNew用のTurnstile検証モック
type mockTurnstileVerifierForNew struct {
	valid bool
	err   error
}

func (m *mockTurnstileVerifierForNew) Verify(ctx context.Context, token string) (bool, error) {
	return m.valid, m.err
}

func TestNew(t *testing.T) {
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
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(queries)

	// ユースケースを初期化
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifierForNew{valid: true, err: nil}

	handler := sign_in.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		userPasswordRepo,
		userSessionRepo,
		userTwoFactorAuthRepo,
		createUserSessionUC,
		mockTurnstile,
	)

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// レスポンスボディを検証
	body := rr.Body.String()

	// ログインフォームが含まれているか確認
	if !strings.Contains(body, `action="/sign_in"`) {
		t.Error("login form action not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	// メールアドレス入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="email"`) {
		t.Error("email input field not found in response")
	}

	// パスワード入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="password"`) {
		t.Error("password input field not found in response")
	}

	// Turnstileウィジェットが含まれているか確認
	if !strings.Contains(body, "test-site-key") {
		t.Error("Turnstile site key not found in response")
	}

	// backパラメータ用のhiddenフィールドが含まれているか確認
	if !strings.Contains(body, `name="back"`) {
		t.Error("back hidden field not found in response")
	}
}

func TestNew_WithBackParameter(t *testing.T) {
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
		TurnstileSiteKey:   "test-site-key",
		TurnstileSecretKey: "",
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(queries)

	// ユースケースを初期化
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// モックTurnstileを初期化
	mockTurnstile := &mockTurnstileVerifierForNew{valid: true, err: nil}

	handler := sign_in.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		userRepo,
		userPasswordRepo,
		userSessionRepo,
		userTwoFactorAuthRepo,
		createUserSessionUC,
		mockTurnstile,
	)

	tests := []struct {
		name       string
		backURL    string
		wantInBody string
	}{
		{
			name:       "backパラメータあり",
			backURL:    "/oauth/authorize?client_id=xxx",
			wantInBody: `name="back" value="/oauth/authorize?client_id=xxx"`,
		},
		{
			name:       "backパラメータなし",
			backURL:    "",
			wantInBody: `name="back" value=""`,
		},
		{
			name:       "日本語パスのbackパラメータ",
			backURL:    "/users/テスト",
			wantInBody: `name="back" value="/users/テスト"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// HTTPリクエストを作成
			targetURL := "/sign_in"
			if tt.backURL != "" {
				targetURL = "/sign_in?back=" + tt.backURL
			}
			req := httptest.NewRequest(http.MethodGet, targetURL, nil)
			req.Header.Set("Accept-Language", "ja")

			// CSRFトークンをコンテキストに設定
			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.New(rr, req)

			// ステータスコードを検証
			if rr.Code != http.StatusOK {
				t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			// backパラメータがhiddenフィールドに含まれているか確認
			body := rr.Body.String()
			if !strings.Contains(body, tt.wantInBody) {
				t.Errorf("backパラメータがhiddenフィールドに含まれていません\nwant: %s", tt.wantInBody)
			}
		})
	}
}
