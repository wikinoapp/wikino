package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestReverseProxyMiddleware_isGoHandledPath(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Domain: "wikino.app",
	}

	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)
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

		// Go版で処理するパス（完全一致）
		{
			name:     "トップページ",
			path:     "/",
			expected: true,
		},

		// Rails版にプロキシするパス
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

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
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

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
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

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
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
	m, err := NewReverseProxyMiddleware("http://localhost:99999", cfg, nil)
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

func TestReverseProxyMiddleware_getFeatureFlagForRequest(t *testing.T) {
	// グローバル変数 featureFlaggedPatterns を変更するため t.Parallel() は使用しない

	cfg := &config.Config{
		Domain: "wikino.app",
	}

	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddleware failed: %v", err)
	}

	// テスト用のパターンを一時的に設定
	originalPatterns := featureFlaggedPatterns
	featureFlaggedPatterns = []featureFlaggedPattern{
		{
			pattern: regexp.MustCompile(`^/@[^/]+/[^/]+/pages/[^/]+$`),
			flag:    "go_page_show",
		},
		{
			pattern: regexp.MustCompile(`^/settings$`),
			flag:    "go_settings",
		},
		{
			pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+/edit$`),
			flag:    "go_page_edit",
		},
		{
			pattern: regexp.MustCompile(`^/s/[^/]+/pages/\d+$`),
			flag:    "go_page_edit",
			methods: []string{"PATCH"},
		},
	}
	defer func() { featureFlaggedPatterns = originalPatterns }()

	testCases := []struct {
		name     string
		method   string
		path     string
		expected model.FeatureFlagName
	}{
		{
			name:     "マッチするパス（ページ表示）",
			method:   http.MethodGet,
			path:     "/@username/space_atname/pages/abc123",
			expected: "go_page_show",
		},
		{
			name:     "マッチするパス（設定）",
			method:   http.MethodGet,
			path:     "/settings",
			expected: "go_settings",
		},
		{
			name:     "マッチしないパス",
			method:   http.MethodGet,
			path:     "/timeline",
			expected: "",
		},
		{
			name:     "部分一致しないパス",
			method:   http.MethodGet,
			path:     "/settings/profile",
			expected: "",
		},
		{
			name:     "ページ編集画面（GET）",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/edit",
			expected: "go_page_edit",
		},
		{
			name:     "ページ更新（PATCH）",
			method:   http.MethodPatch,
			path:     "/s/my-space/pages/1",
			expected: "go_page_edit",
		},
		{
			name:     "ページ表示（GET）はmethodsフィルタによりマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			result := m.getFeatureFlagForRequest(req)
			if result != tc.expected {
				t.Errorf("getFeatureFlagForRequest(%s %q) = %q, want %q", tc.method, tc.path, result, tc.expected)
			}
		})
	}
}

func TestReverseProxyMiddleware_Middleware_FeatureFlag(t *testing.T) {
	// グローバル変数 featureFlaggedPatterns を変更するため t.Parallel() は使用しない

	_, tx := testutil.SetupTx(t)

	// テスト用ユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).Build()

	// テスト用セッションを作成
	sessionToken := testutil.NewSessionBuilder(t, tx).
		WithUserID(userID).
		WithToken("test-feature-flag-token").
		BuildAndGetToken()

	// フィーチャーフラグを作成
	testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(userID).
		WithName("go_settings").
		Build()

	// テスト用のパターンを一時的に設定
	originalPatterns := featureFlaggedPatterns
	featureFlaggedPatterns = []featureFlaggedPattern{
		{
			pattern: regexp.MustCompile(`^/settings$`),
			flag:    "go_settings",
		},
	}
	defer func() { featureFlaggedPatterns = originalPatterns }()

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

	// FeatureFlagRepositoryをトランザクション内で作成
	queries := query.New(testutil.GetTestDB())
	featureFlagRepo := repository.NewFeatureFlagRepository(queries).WithTx(tx)

	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, featureFlagRepo)
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

	t.Run("フラグが有効なユーザーはGo版で処理される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.AddCookie(&http.Cookie{
			Name:  session.CookieName,
			Value: sessionToken,
		})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Go-Handled") != "true" {
			t.Error("フラグが有効なユーザーのリクエストがGo版で処理されなかった")
		}
		if rr.Body.String() != "Go response" {
			t.Errorf("レスポンスが期待と異なる: got %q want %q", rr.Body.String(), "Go response")
		}
	})

	t.Run("フラグが無効なユーザーはRails版に転送される", func(t *testing.T) {
		// フラグが設定されていない別のユーザーを作成
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("other@example.com").
			WithAtname("other_user").
			Build()
		otherToken := testutil.NewSessionBuilder(t, tx).
			WithUserID(otherUserID).
			WithToken("other-session-token").
			BuildAndGetToken()

		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.AddCookie(&http.Cookie{
			Name:  session.CookieName,
			Value: otherToken,
		})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Rails-Handled") != "true" {
			t.Error("フラグが無効なユーザーのリクエストがRails版に転送されなかった")
		}
		if rr.Body.String() != "Rails response" {
			t.Errorf("レスポンスが期待と異なる: got %q want %q", rr.Body.String(), "Rails response")
		}
	})

	t.Run("Cookieがない場合はRails版に転送される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Rails-Handled") != "true" {
			t.Error("Cookieがないリクエストがrails版に転送されなかった")
		}
	})

	t.Run("空のCookie値の場合はRails版に転送される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.AddCookie(&http.Cookie{
			Name:  session.CookieName,
			Value: "",
		})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Rails-Handled") != "true" {
			t.Error("空のCookieのリクエストがRails版に転送されなかった")
		}
	})
}

func TestReverseProxyMiddleware_Middleware_FeatureFlag_NilRepo(t *testing.T) {
	// グローバル変数 featureFlaggedPatterns を変更するため t.Parallel() は使用しない

	// テスト用のパターンを一時的に設定
	originalPatterns := featureFlaggedPatterns
	featureFlaggedPatterns = []featureFlaggedPattern{
		{
			pattern: regexp.MustCompile(`^/settings$`),
			flag:    "go_settings",
		},
	}
	defer func() { featureFlaggedPatterns = originalPatterns }()

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

	// featureFlagRepoをnilで作成
	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddleware failed: %v", err)
	}

	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Go-Handled", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})

	handler := m.Middleware(goHandler)

	// featureFlagRepoがnilの場合、フラグパターンにマッチしてもRails版に転送される
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "some-token",
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Rails-Handled") != "true" {
		t.Error("featureFlagRepoがnilの場合、リクエストがRails版に転送されるべき")
	}
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
