package manifest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/manifest"
)

func TestShow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		env          string
		wantName     string
		wantStatus   int
		wantContents []string
	}{
		{
			name:       "本番環境ではアプリ名がWikinoになる",
			env:        "prod",
			wantName:   "Wikino",
			wantStatus: http.StatusOK,
		},
		{
			name:       "テスト環境ではアプリ名がWikinoになる",
			env:        "test",
			wantName:   "Wikino",
			wantStatus: http.StatusOK,
		},
		{
			name:       "開発環境ではアプリ名がWikino (Dev)になる",
			env:        "dev",
			wantName:   "Wikino (Dev)",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				Env:             tt.env,
				Port:            "8080",
				Domain:          "localhost",
				CookieDomain:    "",
				SessionSecure:   false,
				SessionHTTPOnly: true,
			}

			handler := manifest.NewHandler(cfg)

			req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
			req.Header.Set("Accept-Language", "ja")

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			// ステータスコードを検証
			if rr.Code != tt.wantStatus {
				t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, tt.wantStatus)
			}

			// Content-Typeを検証
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/manifest+json" {
				t.Errorf("Content-Type = %v、期待値 = %v", contentType, "application/manifest+json")
			}

			// JSONをパースして検証
			var m manifest.Manifest
			if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
				t.Fatalf("レスポンスのデコードに失敗: %v", err)
			}

			// アプリ名を検証
			if m.Name != tt.wantName {
				t.Errorf("name = %v、期待値 = %v", m.Name, tt.wantName)
			}

			// 固定値を検証
			if m.ShortName != "Wikino" {
				t.Errorf("short_name = %v、期待値 = %v", m.ShortName, "Wikino")
			}
			if m.Display != "standalone" {
				t.Errorf("display = %v、期待値 = %v", m.Display, "standalone")
			}
			if m.StartURL != "/" {
				t.Errorf("start_url = %v、期待値 = %v", m.StartURL, "/")
			}
			if m.Scope != "/" {
				t.Errorf("scope = %v、期待値 = %v", m.Scope, "/")
			}
			if m.BackgroundColor != "#ffffff" {
				t.Errorf("background_color = %v、期待値 = %v", m.BackgroundColor, "#ffffff")
			}
			if m.ThemeColor != "#ffffff" {
				t.Errorf("theme_color = %v、期待値 = %v", m.ThemeColor, "#ffffff")
			}

			// アイコンを検証
			if len(m.Icons) != 2 {
				t.Errorf("アイコンの件数 = %v、期待値 = %v", len(m.Icons), 2)
			}

			// 192x192アイコンを検証
			if m.Icons[0].Sizes != "192x192" {
				t.Errorf("アイコンのサイズ = %v、期待値 = %v", m.Icons[0].Sizes, "192x192")
			}
			if m.Icons[0].Type != "image/png" {
				t.Errorf("アイコンのtype = %v、期待値 = %v", m.Icons[0].Type, "image/png")
			}

			// 512x512アイコンを検証
			if m.Icons[1].Sizes != "512x512" {
				t.Errorf("アイコンのサイズ = %v、期待値 = %v", m.Icons[1].Sizes, "512x512")
			}
		})
	}
}
