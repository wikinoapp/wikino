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
