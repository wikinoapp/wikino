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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	// レスポンスボディを検証
	body := rr.Body.String()

	// ヒーローセクションが含まれているか確認
	if !strings.Contains(body, "sign_up") {
		t.Error("レスポンスにサインアップのリンクが見つからない")
	}

	// サインインリンクが含まれているか確認
	if !strings.Contains(body, "sign_in") {
		t.Error("レスポンスにサインインのリンクが見つからない")
	}

	// 機能紹介セクションの画像が含まれているか確認
	if !strings.Contains(body, "/static/images/welcome/feature_1.png") {
		t.Error("レスポンスに機能紹介の画像が見つからない")
	}

	// トップページはグローバルナビの対象外である。ナビ項目がヒーローとフッターのCTAと重複し、
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
			t.Errorf("トップページに%qが描画されている", notWant)
		}
	}

	// mainランドマークは残る。ナビの部品ではなくページの主要領域だからである。
	if !strings.Contains(body, `<main id="main" tabindex="-1">`) {
		t.Error("トップページにmainのランドマークが無い")
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

	// コンテキストにユーザー情報を設定 (ログイン状態をシミュレート)
	user := &model.User{
		ID:     "test-user-id",
		Atname: "testuser",
	}
	ctx := middleware.SetUserToContext(req.Context(), user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	// ステータスコードを検証 (リダイレクト)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusSeeOther)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/home" {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, "/home")
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
				t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			// レスポンスボディを検証
			body := rr.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
		})
	}
}

// 公開トップページはインデックス対象のため、自身の絶対アドレスを正規URLとして宣言する。共通
// headに空の値を出させると、リクエストされたURLに解決されてしまう。
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
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		`<link rel="canonical" href="https://localhost/">`,
		`<meta property="og:url" content="https://localhost/">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
}
