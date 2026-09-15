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
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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

		// Go版で処理するパス (完全一致)
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
				t.Errorf("isGoHandledPath(%q) = %v、期待値 = %v", tc.path, result, tc.expected)
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
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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
		t.Errorf("レスポンス = %q、期待値 = %q", rr.Body.String(), "Go response")
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
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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
		t.Errorf("レスポンス = %q、期待値 = %q", rr.Body.String(), "Rails response")
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
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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
			t.Errorf("X-Forwarded-Proto = %q、期待値 = %q", receivedHeaders.Get("X-Forwarded-Proto"), "https")
		}

		// X-Forwarded-Hostが設定されることを確認
		if receivedHeaders.Get("X-Forwarded-Host") != "wikino.app" {
			t.Errorf("X-Forwarded-Host = %q、期待値 = %q", receivedHeaders.Get("X-Forwarded-Host"), "wikino.app")
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
			t.Errorf("X-Real-IP = %q、期待値 = %q", receivedHeaders.Get("X-Real-IP"), "203.0.113.1")
		}
	})

	t.Run("既存のX-Forwarded-Forがある場合は維持される", func(t *testing.T) {
		// Cloudflareなどの上流プロキシがX-Forwarded-Forを設定するシナリオを想定
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.2")
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// 既存のX-Forwarded-Forがそのまま維持されることを確認
		if got := receivedHeaders.Get("X-Forwarded-For"); got != "203.0.113.1, 198.51.100.2" {
			t.Errorf("X-Forwarded-For = %q、期待値 = %q", got, "203.0.113.1, 198.51.100.2")
		}
	})

	t.Run("X-Forwarded-Forがない場合はclientIPが設定される", func(t *testing.T) {
		// クライアントが直接接続するシナリオを想定
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// RemoteAddr由来のIPがX-Forwarded-Forに設定されることを確認
		if got := receivedHeaders.Get("X-Forwarded-For"); got != "192.168.1.1" {
			t.Errorf("X-Forwarded-For = %q、期待値 = %q", got, "192.168.1.1")
		}
	})

	t.Run("X-Forwarded-ForがなくCF-Connecting-IPがある場合はCF-Connecting-IPが使われる", func(t *testing.T) {
		// Cloudflare経由だがX-Forwarded-Forは未設定のシナリオを想定
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.Header.Set("CF-Connecting-IP", "203.0.113.1")
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// CF-Connecting-IPの値がX-Forwarded-Forに設定されることを確認
		if got := receivedHeaders.Get("X-Forwarded-For"); got != "203.0.113.1" {
			t.Errorf("X-Forwarded-For = %q、期待値 = %q", got, "203.0.113.1")
		}
	})

	t.Run("クライアントのHostヘッダがそのままRailsに転送される", func(t *testing.T) {
		// pr.SetURL(parsedURL) の後にpr.Out.Host = pr.In.Hostを設定する
		// 挙動が効いているかを退行テストとして検証する
		req := httptest.NewRequest(http.MethodGet, "/settings", nil)
		req.Host = "example.com"
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if receivedHost != "example.com" {
			t.Errorf("Host = %q、期待値 = %q (pr.Out.Host = pr.In.Hostが効いているか)", receivedHost, "example.com")
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
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
	}

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("リクエストはRails版に転送されるべき")
	}))

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// 502 Bad Gatewayが返されることを確認
	if rr.Code != http.StatusBadGateway {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusBadGateway)
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
	// グローバル変数featureFlaggedPatternsを変更するためt.Parallel() は使用しない

	cfg := &config.Config{
		Domain: "wikino.app",
	}

	m, err := NewReverseProxyMiddleware("http://localhost:3000", cfg, nil)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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
				t.Errorf("getFeatureFlagForRequest(%s %q) = %q、期待値 = %q", tc.method, tc.path, result, tc.expected)
			}
		})
	}
}

func TestReverseProxyMiddleware_Middleware_FeatureFlag(t *testing.T) {
	// グローバル変数featureFlaggedPatternsを変更するためt.Parallel() は使用しない

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
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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
			t.Errorf("レスポンス = %q、期待値 = %q", rr.Body.String(), "Go response")
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
			t.Errorf("レスポンス = %q、期待値 = %q", rr.Body.String(), "Go response")
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
			t.Errorf("レスポンス = %q、期待値 = %q", rr.Body.String(), "Rails response")
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
	// グローバル変数featureFlaggedPatternsを変更するためt.Parallel() は使用しない

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
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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
			name:     "トピック作成フォーム (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space/topics/new",
			expected: true,
		},
		{
			name:     "トピック作成フォーム (HEAD)",
			method:   http.MethodHead,
			path:     "/s/my-space/topics/new",
			expected: true,
		},
		{
			name:     "トピック作成 (POST)",
			method:   http.MethodPost,
			path:     "/s/my-space/topics",
			expected: true,
		},
		{
			name:     "トピックの一般設定 (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space/topics/1/settings/general",
			expected: true,
		},
		{
			name:     "トピックの一般設定 (HEAD)",
			method:   http.MethodHead,
			path:     "/s/my-space/topics/1/settings/general",
			expected: true,
		},
		{
			name:     "トピックの一般設定の保存 (PATCH)",
			method:   http.MethodPatch,
			path:     "/s/my-space/topics/1/settings/general",
			expected: true,
		},
		{
			name:     "トピックの一般設定の保存 (Method Override前のPOST)",
			method:   http.MethodPost,
			path:     "/s/my-space/topics/1/settings/general",
			expected: true,
		},
		{
			name:     "トピック設定のトップはGo版で処理しない",
			method:   http.MethodGet,
			path:     "/s/my-space/topics/1/settings",
			expected: false,
		},
		{
			name:     "トピック削除フォームはGo版で処理しない",
			method:   http.MethodGet,
			path:     "/s/my-space/topics/1/settings/deletion/new",
			expected: false,
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
			// 本操作は常にGoが描画するページ表示画面から呼ばれるため、POSTもGoで処理する。
			name:     "ゴミ箱へ入れる (POST)",
			method:   http.MethodPost,
			path:     "/s/my-space/pages/1/trash",
			expected: true,
		},
		{
			name:     "og:imageエンドポイント (GET)",
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
			// Rails版にこのパスのGETルートは無いため、GETはGoハンドラーが応答せず
			// Railsに転送されてRoutingErrorになるべき。
			name:     "ゴミ箱へ入れる (GET) はPOST限定パターンにマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/pages/1/trash",
			expected: false,
		},
		{
			// 末尾 $ によりサブパスは対象外にし、将来 /trash/... のルートが増えても本パターンが
			// 巻き込まないようにする。
			name:     "ゴミ箱配下のサブパスはマッチしない",
			method:   http.MethodPost,
			path:     "/s/my-space/pages/1/trash/restore",
			expected: false,
		},
		{
			// スペース単位のゴミ箱画面 (/s/:identifier/trash) はRails版のまま残るため、
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
			name:     "og:imageエンドポイント (POST) はGETのみフィルタによりマッチしない",
			method:   http.MethodPost,
			path:     "/attachments/01HXYZ123/og_image",
			expected: false,
		},
		{
			name:     "og:imageエンドポイント (PATCH) はGETのみフィルタによりマッチしない",
			method:   http.MethodPatch,
			path:     "/attachments/01HXYZ123/og_image",
			expected: false,
		},
		{
			name:     "og:imageの末尾セグメントがないパスはマッチしない",
			method:   http.MethodGet,
			path:     "/attachments/01HXYZ123",
			expected: false,
		},
		{
			name:     "og:imageの末尾に余分なセグメントがあるとマッチしない",
			method:   http.MethodGet,
			path:     "/attachments/01HXYZ123/og_image/extra",
			expected: false,
		},
		{
			// GoルートはこのパスのGETだけを処理するため、他のメソッドはGoのルーターへ
			// 届かせない。Rails側へフォールスルーさせ、通常の未一致ルート処理に応答を委ねる。
			name:     "ページ新規作成の入口 (POST) はGETのみフィルタによりマッチしない",
			method:   http.MethodPost,
			path:     "/s/my-space/topics/1/pages/new",
			expected: false,
		},
		{
			// 末尾 $ によりサブパスは対象外にし、将来 /pages/new/... のルートが増えても
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
			// Goルートはトピック一覧のPOSTだけに応答するため、そのGETはRailsへ
			// フォールスルーする。スペースのトピックを並べる画面は今もRails側にある。
			name:     "トピック一覧 (GET) はPOSTのみフィルタによりマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/topics",
			expected: false,
		},
		{
			name:     "トピック作成フォーム配下のサブパスはマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/topics/new/extra",
			expected: false,
		},
		{
			name:     "エクスポート開始画面 (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space/settings/exports/new",
			expected: true,
		},
		{
			name:     "エクスポート開始 (POST)",
			method:   http.MethodPost,
			path:     "/s/my-space/settings/exports",
			expected: true,
		},
		{
			name:     "エクスポート状態表示 (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space/settings/exports/0198f3a0-1b2c-7d3e-8f40-a1b2c3d4e5f6",
			expected: true,
		},
		{
			name:     "エクスポートのダウンロード (GET)",
			method:   http.MethodGet,
			path:     "/s/my-space/settings/exports/0198f3a0-1b2c-7d3e-8f40-a1b2c3d4e5f6/download",
			expected: true,
		},
		{
			name:     "エクスポート一覧もGoルーターで拒否する",
			method:   http.MethodGet,
			path:     "/s/my-space/settings/exports",
			expected: true,
		},
		{
			// スペース設定の残りはRails版のままなので、パターンが設定のパス自体を巻き込んでは
			// ならない。
			name:     "スペース設定はマッチしない",
			method:   http.MethodGet,
			path:     "/s/my-space/settings",
			expected: false,
		},
		{
			name:     "不正なエクスポートIDもGoルーターで拒否する",
			method:   http.MethodGet,
			path:     "/s/my-space/settings/exports/not-a-uuid",
			expected: true,
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
				t.Errorf("isGoHandledByRegex(%s %q) = %v、期待値 = %v", tc.method, tc.path, result, tc.expected)
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
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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
			t.Errorf("SameSite = %v、期待値 = %v", deviceCookie.SameSite, http.SameSiteLaxMode)
		}

		if deviceCookie.Domain != "wikino.app" {
			t.Errorf("Domain = %q、期待値 = %q", deviceCookie.Domain, "wikino.app")
		}

		expectedMaxAge := 10 * 365 * 24 * 60 * 60
		if deviceCookie.MaxAge != expectedMaxAge {
			t.Errorf("MaxAge = %d、期待値 = %d", deviceCookie.MaxAge, expectedMaxAge)
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
	// 常にGoで処理するパス判定を通り抜けたリクエストに対してミドルウェアが
	// グローバルのfeatureFlaggedPatternsを読むため、このグローバル変数を
	// 上書きする他テストと並行実行されないようt.Parallel() は意図的に使用しない。

	// Rails版をモックするテストサーバー
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

	// featureFlagRepoはnilとする。本テストの対象はフラグ判定ではなく
	// device_tokenの発行位置のため。
	m, err := NewReverseProxyMiddleware(railsServer.URL, cfg, nil)
	if err != nil {
		t.Fatalf("NewReverseProxyMiddlewareに失敗: %v", err)
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
		// 常にGoで処理するパス: ensureDeviceTokenはGoパス判定の後でのみ
		// 走るためdevice_tokenは発行されない
		{name: "静的アセットには発行しない", method: http.MethodGet, path: "/static/css/app.css", wantDeviceCookie: false},
		{name: "ヘルスチェックには発行しない", method: http.MethodGet, path: "/health", wantDeviceCookie: false},
		{name: "ホーム画面 (完全一致) には発行しない", method: http.MethodGet, path: "/home", wantDeviceCookie: false},
		{name: "正規表現マッチのGoパスには発行しない", method: http.MethodGet, path: "/s/my-space/pages/1/edit", wantDeviceCookie: false},
		{name: "常時Go化されたスペース詳細には発行しない", method: http.MethodGet, path: "/s/my-space", wantDeviceCookie: false},
		{name: "常時Go化されたページ表示には発行しない", method: http.MethodGet, path: "/s/my-space/pages/1", wantDeviceCookie: false},

		// Rails転送パス: device_tokenが発行される
		{name: "Rails転送パスには発行する", method: http.MethodGet, path: "/settings", wantDeviceCookie: true},
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
				t.Errorf("device_token Cookieの発行 = %v、期待値 = %v (path: %q)", gotDeviceCookie, tc.wantDeviceCookie, tc.path)
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
			t.Errorf("HTMLに%qが含まれていない", expected)
		}
	}
}

func TestReverseProxyMiddleware_ExportNamespace(t *testing.T) {
	t.Parallel()
	railsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Rails-Handled", "true")
		w.WriteHeader(http.StatusOK)
	}))
	defer railsServer.Close()
	m, err := NewReverseProxyMiddleware(railsServer.URL, &config.Config{Domain: "wikino.app"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 未対応のエクスポートURLはGoルーターの拒否へ進み、Railsの書き込み処理には到達させない。
	h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Go-Handled", "true")
		http.NotFound(w, r)
	}))
	for _, tc := range []struct {
		path  string
		rails bool
	}{
		{path: "/s/demo/settings/exports.json"},
		{path: "/s/demo/settings/exports.html"},
		{path: "/s/demo/settings/exports/"},
		{path: "/s/demo/settings/exports/new.json"},
		{path: "/s/demo/settings/exports/not-a-uuid"},
		{path: "/s/demo/settings/exports/0198f3a0-1b2c-7d3e-8f40-a1b2c3d4e5f6.json"},
		{path: "/s/demo/settings/exports/0198f3a0-1b2c-7d3e-8f40-a1b2c3d4e5f6/download.json"},
		{path: "/s//demo//settings//exports.json"},
		{path: "/s/demo/settings", rails: true},
		{path: "/s/demo/settings/deletion", rails: true},
		{path: "/s/demo/settings/exports-other", rails: true},
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
			t.Run(method+" "+tc.path, func(t *testing.T) {
				rr := httptest.NewRecorder()
				h.ServeHTTP(rr, httptest.NewRequest(method, tc.path, nil))
				if tc.rails {
					if rr.Header().Get("X-Rails-Handled") != "true" {
						t.Error("他のスペース設定がRailsへ転送されていません")
					}
				} else if rr.Code != http.StatusNotFound || rr.Header().Get("X-Go-Handled") != "true" || rr.Header().Get("X-Rails-Handled") != "" {
					t.Errorf("ステータス = %d、ヘッダー = %v", rr.Code, rr.Header())
				}
			})
		}
	}
}
