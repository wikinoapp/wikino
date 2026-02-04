package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
)

func TestReverseProxyMiddleware_isGoHandledPath(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Domain: "wikino.app",
	}

	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddleware failed: %v", err)
	}

	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		// Go版で処理するパス
		{
			name:     "静的ファイル",
			path:     "/static/css/app.css",
			expected: true,
		},
		{
			name:     "ヘルスチェック",
			path:     "/health",
			expected: true,
		},
		{
			name:     "マニフェスト",
			path:     "/manifest.json",
			expected: true,
		},
		{
			name:     "ログインページ",
			path:     "/sign_in",
			expected: true,
		},
		{
			name:     "セッション作成",
			path:     "/user_session",
			expected: true,
		},
		{
			name:     "2FAコード入力",
			path:     "/sign_in/two_factor/new",
			expected: true,
		},
		{
			name:     "2FAコード検証",
			path:     "/sign_in/two_factor",
			expected: true,
		},
		{
			name:     "リカバリーコード入力",
			path:     "/sign_in/two_factor/recovery/new",
			expected: true,
		},
		{
			name:     "リカバリーコード検証",
			path:     "/sign_in/two_factor/recovery",
			expected: true,
		},
		{
			name:     "サインアップページ",
			path:     "/sign_up",
			expected: true,
		},
		{
			name:     "メール確認コード送信",
			path:     "/email_confirmation",
			expected: true,
		},
		{
			name:     "メール確認コード入力フォーム",
			path:     "/email_confirmation/edit",
			expected: true,
		},
		{
			name:     "アカウント作成フォーム",
			path:     "/accounts/new",
			expected: true,
		},
		{
			name:     "アカウント作成",
			path:     "/accounts",
			expected: true,
		},

		// Rails版にプロキシするパス
		// トップページはGo版で未実装のため、Rails版にプロキシ
		// Go版で実装後は goHandledExactPaths に "/" を追加する
		{
			name:     "トップページ",
			path:     "/",
			expected: false,
		},
		// 完全一致の "/" がプレフィックス一致として動作しないことを確認
		{
			name:     "ユーザープロフィール",
			path:     "/@username",
			expected: false,
		},
		{
			name:     "スペースページ",
			path:     "/@username/space_atname",
			expected: false,
		},
		{
			name:     "ページ",
			path:     "/@username/space_atname/pages/abc123",
			expected: false,
		},
		{
			name:     "設定ページ",
			path:     "/settings",
			expected: false,
		},
		{
			name:     "作品一覧ページ（/worksは/のプレフィックスだがRails版）",
			path:     "/works",
			expected: false,
		},
		{
			name:     "タイムラインページ（/timelineは/のプレフィックスだがRails版）",
			path:     "/timeline",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := m.isGoHandledPath(tc.path)
			if result != tc.expected {
				t.Errorf("isGoHandledPath(%q) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}

func TestReverseProxyMiddleware_Middleware_GoPath(t *testing.T) {
	t.Parallel()

	// Rails版をモックするテストサーバー
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rails-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	cfg := &config.Config{
		Domain: "wikino.app",
	}

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddleware failed: %v", err)
	}

	// Go版のハンドラー
	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Go-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})

	handler := m.Middleware(goHandler)

	req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Go-Handled") != "true" {
		t.Error("Go版で処理されるべきリクエストがRails版に転送された")
	}

	if rr.Body.String() != "Go response" {
		t.Errorf("レスポンスが期待と異なる: got %q want %q", rr.Body.String(), "Go response")
	}
}

func TestReverseProxyMiddleware_Middleware_RailsPath(t *testing.T) {
	t.Parallel()

	// Rails版をモックするテストサーバー
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rails-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	cfg := &config.Config{
		Domain: "wikino.app",
	}

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddleware failed: %v", err)
	}

	// Go版のハンドラー
	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Go-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})

	handler := m.Middleware(goHandler)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Rails-Handled") != "true" {
		t.Error("Rails版に転送されるべきリクエストがGo版で処理された")
	}

	if rr.Body.String() != "Rails response" {
		t.Errorf("レスポンスが期待と異なる: got %q want %q", rr.Body.String(), "Rails response")
	}
}

func TestReverseProxyMiddleware_ProxyHeaders(t *testing.T) {
	t.Parallel()

	var receivedHeaders http.Header
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer railsServer.Close()

	cfg := &config.Config{
		Domain: "wikino.app",
	}

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddleware failed: %v", err)
	}

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("リクエストはRails版に転送されるべき")
	}))

	t.Run("プロキシヘッダーが設定される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// X-Forwarded-Protoが設定されることを確認
		if receivedHeaders.Get("X-Forwarded-Proto") != "https" {
			t.Errorf("X-Forwarded-Proto = %q, want %q", receivedHeaders.Get("X-Forwarded-Proto"), "https")
		}

		// X-Forwarded-Hostが設定されることを確認
		if receivedHeaders.Get("X-Forwarded-Host") != "wikino.app" {
			t.Errorf("X-Forwarded-Host = %q, want %q", receivedHeaders.Get("X-Forwarded-Host"), "wikino.app")
		}
	})

	t.Run("CF-Connecting-IPがある場合はそれを使用", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.Header.Set("CF-Connecting-IP", "203.0.113.1")
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// X-Real-IPがCF-Connecting-IPの値になることを確認
		if receivedHeaders.Get("X-Real-IP") != "203.0.113.1" {
			t.Errorf("X-Real-IP = %q, want %q", receivedHeaders.Get("X-Real-IP"), "203.0.113.1")
		}
	})
}

func TestReverseProxyMiddleware_ErrorHandling(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Domain: "wikino.app",
	}

	// 存在しないURLにプロキシ
	m, err := NewReverseProxyMiddleware("http://localhost:99999", cfg)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddleware failed: %v", err)
	}

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("リクエストはRails版に転送されるべき")
	}))

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// 502 Bad Gatewayが返されることを確認
	if rr.Code != http.StatusBadGateway {
		t.Errorf("ステータスコード = %d, want %d", rr.Code, http.StatusBadGateway)
	}

	// エラーページにWikinoが含まれることを確認
	if !containsString(rr.Body.String(), "Wikino") {
		t.Error("エラーページにWikinoが含まれていない")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRender502ErrorHTML(t *testing.T) {
	t.Parallel()

	html := render502ErrorHTML()

	// HTMLに必要な要素が含まれていることを確認
	expectedStrings := []string{
		"<!DOCTYPE html>",
		"<html lang=\"ja\">",
		"Wikino",
		"サービス接続エラー",
	}

	for _, expected := range expectedStrings {
		if !containsString(html, expected) {
			t.Errorf("HTML should contain %q", expected)
		}
	}
}
