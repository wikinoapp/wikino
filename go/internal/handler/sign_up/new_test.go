package sign_up_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_up"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

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
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)

	handler := sign_up.NewHandler(
		cfg,
		sessionMgr,
	)

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/sign_up", nil)
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

	// サインアップフォームが含まれているか確認
	if !strings.Contains(body, `action="/email_confirmation"`) {
		t.Error("sign up form action not found in response")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("CSRF token not found in response")
	}

	// メールアドレス入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="email"`) {
		t.Error("email input field not found in response")
	}

	// eventのhiddenフィールドが含まれているか確認
	if !strings.Contains(body, `name="event" value="signup"`) {
		t.Error("event hidden field not found in response")
	}

	// Turnstileウィジェットが含まれているか確認
	if !strings.Contains(body, "test-site-key") {
		t.Error("Turnstile site key not found in response")
	}

	// ログインリンクが含まれているか確認
	if !strings.Contains(body, `href="/sign_in"`) {
		t.Error("sign in link not found in response")
	}
}

func TestNew_EnglishLocale(t *testing.T) {
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
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)

	handler := sign_up.NewHandler(
		cfg,
		sessionMgr,
	)

	// HTTPリクエストを作成（英語ロケール）
	req := httptest.NewRequest(http.MethodGet, "/sign_up", nil)
	req.Header.Set("Accept-Language", "en")

	// CSRFトークンと言語設定をコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangEn)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// 英語の見出しが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "Sign up for Wikino") {
		t.Error("English heading not found in response")
	}

	// 英語のボタンテキストが含まれているか確認
	if !strings.Contains(body, "Send confirmation code") {
		t.Error("English submit button text not found in response")
	}
}
