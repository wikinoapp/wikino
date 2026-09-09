package welcome_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/welcome"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
)

func TestShow_未ログイン時にトップページが表示される(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	handler := welcome.NewHandler(cfg, flashMgr)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "ja")

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// レスポンスボディを検証
	body := rr.Body.String()

	// ヒーローセクションが含まれているか確認
	if !strings.Contains(body, "sign_up") {
		t.Error("sign up link not found in response")
	}

	// サインインリンクが含まれているか確認
	if !strings.Contains(body, "sign_in") {
		t.Error("sign in link not found in response")
	}

	// 機能紹介セクションの画像が含まれているか確認
	if !strings.Contains(body, "/static/images/welcome/feature_1.png") {
		t.Error("feature image not found in response")
	}

	// The top page is out of the global navigation: its nav items duplicate the hero and footer calls
	// to action, and it has no breadcrumb items, so the header would carry the bar alone. The header,
	// the bottom bar, the padding that keeps content clear of the bar, and the skip link that exists
	// to bypass the navigation all stay out.
	//
	// [Ja] トップページはグローバルナビの対象外である。ナビ項目がヒーローとフッターの CTA と重複し、
	// パンくず項目も持たないためヘッダーの中身はバーだけになる。ヘッダー・下部バー・バーにコンテンツが
	// 隠れないための余白・ナビを飛ばすためのスキップリンクは、いずれも出さない。
	for _, notWant := range []string{
		`<header class="hidden md:block">`,
		`aria-label="パンくずリスト"`,
		`aria-label="グローバルナビゲーション"`,
		`aria-label="グローバルナビゲーション (モバイル)"`,
		`href="#main"`,
		"pb-[calc(var(--app-bottom-nav-max-height)+0.5rem+env(safe-area-inset-bottom))]",
	} {
		if strings.Contains(body, notWant) {
			t.Errorf("top page should not render %q", notWant)
		}
	}

	// The main landmark stays: it is the page's main region, not a navigation part.
	//
	// [Ja] main ランドマークは残る。ナビの部品ではなくページの主要領域だからである。
	if !strings.Contains(body, `<main id="main" tabindex="-1">`) {
		t.Error("top page should keep the main landmark")
	}
}

func TestShow_ログイン済み時にホームにリダイレクトされる(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	handler := welcome.NewHandler(cfg, flashMgr)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "ja")

	// コンテキストにユーザー情報を設定（ログイン状態をシミュレート）
	user := &model.User{
		ID:     "test-user-id",
		Atname: "testuser",
	}
	ctx := middleware.SetUserToContext(req.Context(), user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	// ステータスコードを検証（リダイレクト）
	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/home" {
		t.Errorf("wrong redirect location: got %v want %v", location, "/home")
	}
}

func TestShow_日本語と英語で正しく表示される(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		acceptLang   string
		wantContains []string
	}{
		{
			name:       "日本語",
			acceptLang: "ja",
			wantContains: []string{
				"/sign_up",
				"/sign_in",
			},
		},
		{
			name:       "英語",
			acceptLang: "en",
			wantContains: []string{
				"/sign_up",
				"/sign_in",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				Env:             "test",
				Port:            "8080",
				Domain:          "localhost",
				CookieDomain:    "",
				SessionSecure:   false,
				SessionHTTPOnly: true,
			}

			flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
			handler := welcome.NewHandler(cfg, flashMgr)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Language", tt.acceptLang)

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			// ステータスコードを検証
			if rr.Code != http.StatusOK {
				t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			// レスポンスボディを検証
			body := rr.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("response doesn't contain %q", want)
				}
			}
		})
	}
}

// The public top page is indexable, so it declares its own absolute address as canonical rather
// than leaving the shared head to emit an empty one that resolves to whatever URL was requested.
//
// [Ja] 公開トップページはインデックス対象のため、自身の絶対アドレスを正規 URL として宣言する。共通
// head に空の値を出させると、リクエストされた URL に解決されてしまう。
func TestShow_CanonicalPointsAtTopPage(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	handler := welcome.NewHandler(cfg, flashMgr)

	req := httptest.NewRequest(http.MethodGet, "/?utm_source=example", nil)
	req.Header.Set("Accept-Language", "ja")

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		`<link rel="canonical" href="https://localhost/">`,
		`<meta property="og:url" content="https://localhost/">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}
}
