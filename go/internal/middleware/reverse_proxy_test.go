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
		{
			name:     "ホーム画面",
			path:     "/home",
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
			name:     "作品一覧ページ (/worksは/のプレフィックスだがRails版)",
			path:     "/works",
			expected: false,
		},
		{
			name:     "タイムラインページ (/timelineは/のプレフィックスだがRails版)",
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
	var receivedHost string
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		receivedHost = r.Host
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

	t.Run("既存のX-Forwarded-Forがある場合は維持される", func(t *testing.T) {
		// Cloudflare などの上流プロキシが X-Forwarded-For を設定するシナリオを想定
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.2")
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// 既存の X-Forwarded-For がそのまま維持されることを確認
		if got := receivedHeaders.Get("X-Forwarded-For"); got != "203.0.113.1, 198.51.100.2" {
			t.Errorf("X-Forwarded-For = %q, want %q", got, "203.0.113.1, 198.51.100.2")
		}
	})

	t.Run("X-Forwarded-Forがない場合はclientIPが設定される", func(t *testing.T) {
		// クライアントが直接接続するシナリオを想定
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// RemoteAddr 由来の IP が X-Forwarded-For に設定されることを確認
		if got := receivedHeaders.Get("X-Forwarded-For"); got != "192.168.1.1" {
			t.Errorf("X-Forwarded-For = %q, want %q", got, "192.168.1.1")
		}
	})

	t.Run("X-Forwarded-ForがなくCF-Connecting-IPがある場合はCF-Connecting-IPが使われる", func(t *testing.T) {
		// Cloudflare 経由だが X-Forwarded-For は未設定のシナリオを想定
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.Header.Set("CF-Connecting-IP", "203.0.113.1")
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// CF-Connecting-IP の値が X-Forwarded-For に設定されることを確認
		if got := receivedHeaders.Get("X-Forwarded-For"); got != "203.0.113.1" {
			t.Errorf("X-Forwarded-For = %q, want %q", got, "203.0.113.1")
		}
	})

	t.Run("クライアントのHostヘッダがそのままRailsに転送される", func(t *testing.T) {
		// pr.SetURL(parsedURL) の後に pr.Out.Host = pr.In.Host を設定する
		// 挙動が効いているかを退行テストとして検証する
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.Host = "example.com"
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if receivedHost != "example.com" {
			t.Errorf("Host = %q, want %q（pr.Out.Host = pr.In.Host が効いているか）", receivedHost, "example.com")
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
			flag:    model.FeatureFlagExample,
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
			name:     "マッチするパス (ページ表示)",
			method:   http.MethodGet,
			path:     "/@username/space_atname/pages/abc123",
			expected: model.FeatureFlagExample,
		},
		{
			name:     "マッチするパス (設定)",
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
			name:     "ページ編集画面 (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/edit",
			expected: "go_page_edit",
		},
		{
			name:     "ページ更新 (PATCH)",
			method:   http.MethodPatch,
			path:     "/s/my-space/pages/1",
			expected: "go_page_edit",
		},
		{
			name:     "ページ更新 (POST) はMethod Override前のためPATCHパターンにマッチする",
			method:   http.MethodPost,
			path:     "/s/my-space/pages/1",
			expected: "go_page_edit",
		},
		{
			name:     "ページ表示 (GET) はmethodsフィルタによりマッチしない",
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

	// ユーザー単位フラグを作成
	testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(userID).
		WithName("go_settings").
		Build()

	// デバイス単位フラグを作成
	deviceToken := "test-device-token-12345"
	testutil.NewFeatureFlagBuilder(t, tx).
		WithDeviceToken(deviceToken).
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

	t.Run("user_idフラグが有効なユーザーはGo版で処理される", func(t *testing.T) {
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

	t.Run("device_tokenフラグが有効なデバイスはGo版で処理される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.AddCookie(&http.Cookie{
			Name:  DeviceTokenCookieName,
			Value: deviceToken,
		})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Go-Handled") != "true" {
			t.Error("device_tokenフラグが有効なリクエストがGo版で処理されなかった")
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

	t.Run("フラグが無効なdevice_tokenはRails版に転送される", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.AddCookie(&http.Cookie{
			Name:  DeviceTokenCookieName,
			Value: "unknown-device-token",
		})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get("X-Rails-Handled") != "true" {
			t.Error("フラグが無効なdevice_tokenのリクエストがRails版に転送されなかった")
		}
	})

	t.Run("両方のCookieがない場合はRails版に転送される", func(t *testing.T) {
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
		req.AddCookie(&http.Cookie{
			Name:  DeviceTokenCookieName,
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

func TestReverseProxyMiddleware_isGoHandledByRegex(t *testing.T) {
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
		method   string
		path     string
		expected bool
	}{
		// Go版で処理するパス
		{
			name:     "スペース詳細 (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space",
			expected: true,
		},
		{
			name:     "スペース配下のトピックパスはスペース詳細パターンではなくトピックパターンにマッチする",
			method:   http.MethodGet,
			path:     "/s/my-space/topics/1",
			expected: true,
		},
		{
			name:     "ページ新規作成の入口 (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space/topics/1/pages/new",
			expected: true,
		},
		{
			name:     "ページ編集画面",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/edit",
			expected: true,
		},
		{
			name:     "ページプレビュー (POST)",
			method:   http.MethodPost,
			path:     "/s/my-space/pages/1/preview",
			expected: true,
		},
		{
			name:     "下書きページ表示",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/draft_page",
			expected: true,
		},
		{
			name:     "下書きページ更新",
			method:   http.MethodPatch,
			path:     "/s/my-space/pages/1/draft_page",
			expected: true,
		},
		{
			name:     "下書きリビジョン更新",
			method:   http.MethodPatch,
			path:     "/s/my-space/pages/1/draft_page_revision",
			expected: true,
		},
		{
			name:     "下書きリビジョン差分 (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/draft_page_revisions/01HXYZ123",
			expected: true,
		},
		{
			name:     "下書きリビジョン復元 (POST)",
			method:   http.MethodPost,
			path:     "/s/my-space/pages/1/draft_page_revisions/01HXYZ123/restore",
			expected: true,
		},
		{
			name:     "ページ表示 (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1",
			expected: true,
		},
		{
			name:     "ページ表示 (HEAD)",
			method:   http.MethodHead,
			path:     "/s/my-space/pages/1",
			expected: true,
		},
		{
			name:     "ページ更新 (PATCH)",
			method:   http.MethodPatch,
			path:     "/s/my-space/pages/1",
			expected: true,
		},
		{
			name:     "ページ更新 (POST→PATCH、Method Override前)",
			method:   http.MethodPost,
			path:     "/s/my-space/pages/1",
			expected: true,
		},
		{
			name:     "ページロケーション一覧",
			method:   http.MethodGet,
			path:     "/s/my-space/page_locations",
			expected: true,
		},
		{
			name:     "リンク一覧",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/link_list",
			expected: true,
		},
		{
			name:     "バックリンク一覧 (個別)",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/links/2/backlink_list",
			expected: true,
		},
		{
			name:     "バックリンク一覧",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/backlinks",
			expected: true,
		},
		{
			name:     "ページ移動",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/move",
			expected: true,
		},
		{
			// The action is reached from the page detail screen, which Go now always serves, so the
			// POST must be handled by Go as well.
			//
			// [Ja] 本操作は常に Go が描画するページ表示画面から呼ばれるため、POST も Go で処理する。
			name:     "ゴミ箱へ入れる (POST)",
			method:   http.MethodPost,
			path:     "/s/my-space/pages/1/trash",
			expected: true,
		},
		{
			name:     "og:image エンドポイント (GET)",
			method:   http.MethodGet,
			path:     "/attachments/01HXYZ123/og_image",
			expected: true,
		},

		// Rails版に転送するパス
		{
			name:     "スペース詳細 (POST) はGETのみフィルタによりマッチしない",
			method:   http.MethodPost,
			path:     "/s/my-space",
			expected: false,
		},
		{
			name:     "スペース識別子の後にスラッシュが続くパスはマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/",
			expected: false,
		},
		{
			// Rails has no GET route for this path, so a GET must fall through to Rails and raise a
			// RoutingError there rather than be answered by the Go handler.
			//
			// [Ja] Rails 版にこのパスの GET ルートは無いため、GET は Go ハンドラーが応答せず
			// Rails に転送されて RoutingError になるべき。
			name:     "ゴミ箱へ入れる (GET) はPOST限定パターンにマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/trash",
			expected: false,
		},
		{
			// The trailing "$" keeps sub-paths out, so a future /trash/... route is not swept in by
			// this pattern.
			//
			// [Ja] 末尾 $ によりサブパスは対象外にし、将来 /trash/... のルートが増えても本パターンが
			// 巻き込まないようにする。
			name:     "ゴミ箱配下のサブパスはマッチしない",
			method:   http.MethodPost,
			path:     "/s/my-space/pages/1/trash/restore",
			expected: false,
		},
		{
			// The space-level trash screen (/s/:identifier/trash) stays on Rails, so it must not be
			// caught by the page-scoped pattern.
			//
			// [Ja] スペース単位のゴミ箱画面 (/s/:identifier/trash) は Rails 版のまま残るため、
			// ページ単位の本パターンが拾ってはいけない。
			name:     "スペースのゴミ箱画面はマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/trash",
			expected: false,
		},
		{
			name:     "ページ番号が数字でないパスはマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/abc",
			expected: false,
		},
		{
			name:     "ページ番号の後にスラッシュが続くパスはマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/",
			expected: false,
		},
		{
			name:     "ページプレビュー (PATCH) はPOSTのみフィルタによりマッチしない",
			method:   http.MethodPatch,
			path:     "/s/my-space/pages/1/preview",
			expected: false,
		},
		{
			name:     "下書きリビジョン差分 (POST) はGETのみフィルタによりマッチしない",
			method:   http.MethodPost,
			path:     "/s/my-space/pages/1/draft_page_revisions/01HXYZ123",
			expected: false,
		},
		{
			name:     "下書きリビジョン復元 (GET) はPOSTのみフィルタによりマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/draft_page_revisions/01HXYZ123/restore",
			expected: false,
		},
		{
			name:     "og:image エンドポイント (POST) はGETのみフィルタによりマッチしない",
			method:   http.MethodPost,
			path:     "/attachments/01HXYZ123/og_image",
			expected: false,
		},
		{
			name:     "og:image エンドポイント (PATCH) はGETのみフィルタによりマッチしない",
			method:   http.MethodPatch,
			path:     "/attachments/01HXYZ123/og_image",
			expected: false,
		},
		{
			name:     "og:image の末尾セグメントがないパスはマッチしない",
			method:   http.MethodGet,
			path:     "/attachments/01HXYZ123",
			expected: false,
		},
		{
			name:     "og:image の末尾に余分なセグメントがあるとマッチしない",
			method:   http.MethodGet,
			path:     "/attachments/01HXYZ123/og_image/extra",
			expected: false,
		},
		{
			// The Go route handles GET alone, so any other method must not reach the Go router.
			// It falls through to Rails and uses its normal unmatched-route handling.
			//
			// [Ja] Go ルートはこのパスの GET だけを処理するため、他のメソッドは Go のルーターへ
			// 届かせない。Rails 側へフォールスルーさせ、通常の未一致ルート処理に応答を委ねる。
			name:     "ページ新規作成の入口 (POST) はGETのみフィルタによりマッチしない",
			method:   http.MethodPost,
			path:     "/s/my-space/topics/1/pages/new",
			expected: false,
		},
		{
			// The trailing "$" keeps sub-paths out, so a future /pages/new/... route is not swept
			// in by this pattern.
			//
			// [Ja] 末尾 $ によりサブパスは対象外にし、将来 /pages/new/... のルートが増えても
			// 本パターンが巻き込まないようにする。
			name:     "ページ新規作成の入口配下のサブパスはマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/topics/1/pages/new/extra",
			expected: false,
		},
		{
			name:     "トピック番号が数字でないページ新規作成の入口はマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/topics/abc/pages/new",
			expected: false,
		},
		{
			name:     "マッチしないパス",
			method:   http.MethodGet,
			path:     "/settings",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.method, tc.path, nil)
			result := m.isGoHandledByRegex(req)
			if result != tc.expected {
				t.Errorf("isGoHandledByRegex(%s %q) = %v, want %v", tc.method, tc.path, result, tc.expected)
			}
		})
	}
}

func TestReverseProxyMiddleware_ensureDeviceToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Domain:        "wikino.app",
		CookieDomain:  "wikino.app",
		SessionSecure: true,
	}

	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddleware failed: %v", err)
	}

	t.Run("device_token Cookieがない場合は自動生成される", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		m.ensureDeviceToken(rr, req)

		cookies := rr.Result().Cookies()
		var deviceCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == DeviceTokenCookieName {
				deviceCookie = c
				break
			}
		}

		if deviceCookie == nil {
			t.Fatal("device_token Cookieが設定されていない")
		}

		if deviceCookie.Value == "" {
			t.Error("device_token Cookieの値が空")
		}

		if !deviceCookie.HttpOnly {
			t.Error("HttpOnlyが設定されていない")
		}

		if !deviceCookie.Secure {
			t.Error("Secureが設定されていない")
		}

		if deviceCookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("SameSite = %v, want %v", deviceCookie.SameSite, http.SameSiteLaxMode)
		}

		if deviceCookie.Domain != "wikino.app" {
			t.Errorf("Domain = %q, want %q", deviceCookie.Domain, "wikino.app")
		}

		expectedMaxAge := 10 * 365 * 24 * 60 * 60
		if deviceCookie.MaxAge != expectedMaxAge {
			t.Errorf("MaxAge = %d, want %d", deviceCookie.MaxAge, expectedMaxAge)
		}
	})

	t.Run("device_token Cookieが既に存在する場合は生成しない", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  DeviceTokenCookieName,
			Value: "existing-token",
		})
		rr := httptest.NewRecorder()

		m.ensureDeviceToken(rr, req)

		cookies := rr.Result().Cookies()
		for _, c := range cookies {
			if c.Name == DeviceTokenCookieName {
				t.Error("既存のdevice_token Cookieがあるのに新しいCookieが設定された")
			}
		}
	})
}

func TestReverseProxyMiddleware_Middleware_DeviceTokenIssuance(t *testing.T) {
	// The middleware reads the global featureFlaggedPatterns slice for requests
	// that fall through the always-Go checks, so t.Parallel() is intentionally
	// omitted to avoid running concurrently with tests that swap out this
	// global variable.
	//
	// [Ja] 常に Go で処理するパス判定を通り抜けたリクエストに対してミドルウェアが
	// グローバルの featureFlaggedPatterns を読むため、このグローバル変数を
	// 上書きする他テストと並行実行されないよう t.Parallel() は意図的に使用しない。

	// Mock server standing in for the Rails version.
	// [Ja] Rails 版をモックするテストサーバー
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Rails response"))
	}))
	defer railsServer.Close()

	cfg := &config.Config{
		Domain:        "wikino.app",
		CookieDomain:  "wikino.app",
		SessionSecure: true,
	}

	// featureFlagRepo is nil: the subject of this test is the device_token
	// placement, not the flag decision.
	//
	// [Ja] featureFlagRepo は nil とする。本テストの対象はフラグ判定ではなく
	// device_token の発行位置のため。
	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddleware failed: %v", err)
	}

	goHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Go response"))
	})

	handler := m.Middleware(goHandler)

	testCases := []struct {
		name             string
		method           string
		path             string
		wantDeviceCookie bool
	}{
		// Paths always handled by Go: device_token must NOT be issued because
		// ensureDeviceToken runs only after the Go-path checks.
		//
		// [Ja] 常に Go で処理するパス: ensureDeviceToken は Go パス判定の後でのみ
		// 走るため device_token は発行されない
		{name: "静的アセットには発行しない", method: http.MethodGet, path: "/static/css/app.css", wantDeviceCookie: false},
		{name: "ヘルスチェックには発行しない", method: http.MethodGet, path: "/health", wantDeviceCookie: false},
		{name: "ホーム画面 (完全一致) には発行しない", method: http.MethodGet, path: "/home", wantDeviceCookie: false},
		{name: "正規表現マッチの Go パスには発行しない", method: http.MethodGet, path: "/s/my-space/pages/1/edit", wantDeviceCookie: false},
		{name: "常時 Go 化されたスペース詳細には発行しない", method: http.MethodGet, path: "/s/my-space", wantDeviceCookie: false},
		{name: "常時 Go 化されたページ表示には発行しない", method: http.MethodGet, path: "/s/my-space/pages/1", wantDeviceCookie: false},

		// Rails-proxied paths: device_token must be issued.
		// [Ja] Rails 転送パス: device_token が発行される
		{name: "Rails 転送パスには発行する", method: http.MethodGet, path: "/settings", wantDeviceCookie: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			gotDeviceCookie := false
			for _, c := range rr.Result().Cookies() {
				if c.Name == DeviceTokenCookieName {
					gotDeviceCookie = true
					break
				}
			}

			if gotDeviceCookie != tc.wantDeviceCookie {
				t.Errorf("device_token Cookie の発行 = %v, want %v (path: %q)", gotDeviceCookie, tc.wantDeviceCookie, tc.path)
			}
		})
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
