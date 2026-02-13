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
				t.Errorf("wrong status code: got %v want %v", rr.Code, tt.wantStatus)
			}

			// Content-Typeを検証
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/manifest+json" {
				t.Errorf("wrong content type: got %v want %v", contentType, "application/manifest+json")
			}

			// JSONをパースして検証
			var m manifest.Manifest
			if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// アプリ名を検証
			if m.Name != tt.wantName {
				t.Errorf("wrong name: got %v want %v", m.Name, tt.wantName)
			}

			// 固定値を検証
			if m.ShortName != "Wikino" {
				t.Errorf("wrong short_name: got %v want %v", m.ShortName, "Wikino")
			}
			if m.Display != "standalone" {
				t.Errorf("wrong display: got %v want %v", m.Display, "standalone")
			}
			if m.StartURL != "/" {
				t.Errorf("wrong start_url: got %v want %v", m.StartURL, "/")
			}
			if m.Scope != "/" {
				t.Errorf("wrong scope: got %v want %v", m.Scope, "/")
			}
			if m.BackgroundColor != "#ffffff" {
				t.Errorf("wrong background_color: got %v want %v", m.BackgroundColor, "#ffffff")
			}
			if m.ThemeColor != "#ffffff" {
				t.Errorf("wrong theme_color: got %v want %v", m.ThemeColor, "#ffffff")
			}

			// アイコンを検証
			if len(m.Icons) != 2 {
				t.Errorf("wrong number of icons: got %v want %v", len(m.Icons), 2)
			}

			// 192x192アイコンを検証
			if m.Icons[0].Sizes != "192x192" {
				t.Errorf("wrong icon sizes: got %v want %v", m.Icons[0].Sizes, "192x192")
			}
			if m.Icons[0].Type != "image/png" {
				t.Errorf("wrong icon type: got %v want %v", m.Icons[0].Type, "image/png")
			}

			// 512x512アイコンを検証
			if m.Icons[1].Sizes != "512x512" {
				t.Errorf("wrong icon sizes: got %v want %v", m.Icons[1].Sizes, "512x512")
			}
		})
	}
}
