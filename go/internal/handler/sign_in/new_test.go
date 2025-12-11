package sign_in_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/sign_in"
	"github.com/wikinoapp/wikino/go/internal/middleware"
)

func TestNew(t *testing.T) {
	t.Parallel()

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

	handler := sign_in.NewHandler(cfg)

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
	if !strings.Contains(body, `action="/user_session"`) {
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
}
