package sign_up_test

import (
	"database/sql"
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

func setupHandler(t *testing.T, tx *sql.Tx) *sign_up.Handler {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)

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

	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)

	return sign_up.NewHandler(
		cfg,
		sessionMgr,
	)
}

func TestNew(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx)

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
	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx)

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

// The logo link back to the home page is icon-only, so its accessible name comes from the
// translated aria-label rather than from its content.
//
// [Ja] ホームへ戻るロゴリンクはアイコンのみのため、アクセシブルネームは内容ではなく
// 翻訳済みの aria-label が供給する。
func TestNew_LogoLinkHasAccessibleName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		wantLabel string
	}{
		{
			name:      "日本語",
			locale:    i18n.LangJa,
			wantLabel: "Wikino のホーム",
		},
		{
			name:      "英語",
			locale:    i18n.LangEn,
			wantLabel: "Wikino home",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			handler := setupHandler(t, tx)

			req := httptest.NewRequest(http.MethodGet, "/sign_up", nil)

			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			ctx = i18n.SetLocale(ctx, tt.locale)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.New(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			if !strings.Contains(rr.Body.String(), `aria-label="`+tt.wantLabel+`"`) {
				t.Errorf("ロゴリンクに aria-label %q が含まれていない", tt.wantLabel)
			}
		})
	}
}
