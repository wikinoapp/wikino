package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/middleware"
)

func TestCSRFMiddleware_GET(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	// テスト用ハンドラー
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// コンテキストからCSRFトークンを取得
		token := middleware.GetCSRFTokenFromContext(r.Context())
		if token == "" {
			t.Error("GETリクエストでCSRFトークンがコンテキストに設定されていません")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GETリクエストで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}

	// CSRFトークンCookieが設定されていることを確認
	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == middleware.CSRFCookieName {
			csrfCookie = cookie
			break
		}
	}

	if csrfCookie == nil {
		t.Fatal("CSRFトークンCookieが設定されていません")
	}

	if csrfCookie.Value == "" {
		t.Error("CSRFトークンが空です")
	}

	// トークンの長さを確認（Base64エンコード後は44文字）
	if len(csrfCookie.Value) != 44 {
		t.Errorf("CSRFトークンの長さが不正です: got %d want 44", len(csrfCookie.Value))
	}
}

func TestCSRFMiddleware_HEAD(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("HEAD", "/test", nil)
	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("HEADリクエストで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_OPTIONS(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("OPTIONSリクエストで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_POST_ValidToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// 正しいCSRFトークン
	csrfToken := "test_csrf_token_12345678901234567890123456789012"

	// POSTリクエストに正しいCSRFトークンを含める
	form := url.Values{}
	form.Set("csrf_token", csrfToken)

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: csrfToken,
	})

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("正しいCSRFトークンで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("レスポンスが正しくありません: got %v want %v", rr.Body.String(), "OK")
	}
}

func TestCSRFMiddleware_POST_InvalidToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// POSTリクエストに不正なCSRFトークンを含める
	form := url.Values{}
	form.Set("csrf_token", "invalid_token")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: "correct_token",
	})

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("不正なCSRFトークンで403が返されませんでした: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_POST_NoCookie(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// POSTリクエストにCSRFトークンCookieを含めない
	form := url.Values{}
	form.Set("csrf_token", "some_token")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("CSRFクッキーなしで403が返されませんでした: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_POST_NoFormToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// POSTリクエストにフォームトークンを含めない
	form := url.Values{}
	form.Set("other_field", "value")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: "cookie_token",
	})

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("フォームトークンなしで403が返されませんでした: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_POST_HeaderToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	csrfToken := "test_csrf_token_header"

	// POSTリクエストにヘッダーでCSRFトークンを含める
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: csrfToken,
	})

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ヘッダーのCSRFトークンで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("レスポンスが正しくありません: got %v want %v", rr.Body.String(), "OK")
	}
}

func TestCSRFMiddleware_PATCH(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	csrfToken := "test_csrf_token_patch"

	form := url.Values{}
	form.Set("csrf_token", csrfToken)

	req := httptest.NewRequest("PATCH", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: csrfToken,
	})

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("PATCHリクエストで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_DELETE(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	csrfToken := "test_csrf_token_delete"

	// DELETEリクエストではヘッダーでCSRFトークンを送る
	req := httptest.NewRequest("DELETE", "/test", nil)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: csrfToken,
	})

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("DELETEリクエストで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_PUT(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	csrfToken := "test_csrf_token_put"

	form := url.Values{}
	form.Set("csrf_token", csrfToken)

	req := httptest.NewRequest("PUT", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: csrfToken,
	})

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("PUTリクエストで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_ExistingCookie(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	existingToken := "existing_csrf_token_12345"

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := middleware.GetCSRFTokenFromContext(r.Context())
		if token != existingToken {
			t.Errorf("既存のトークンが使用されていません: got %v want %v", token, existingToken)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: existingToken,
	})
	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("既存のトークンでGETリクエストが失敗しました: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_CookieAttributes(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   true,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == middleware.CSRFCookieName {
			csrfCookie = cookie
			break
		}
	}

	if csrfCookie == nil {
		t.Fatal("CSRFクッキーが設定されていません")
	}

	// Cookie属性を確認
	// http.CookieはSet-Cookieヘッダーからのパース時にドメインの先頭のドットを削除する場合がある
	if csrfCookie.Domain != ".example.com" && csrfCookie.Domain != "example.com" {
		t.Errorf("Domainが正しくありません: got %v want .example.com or example.com", csrfCookie.Domain)
	}

	if !csrfCookie.Secure {
		t.Error("Secure属性がtrueではありません")
	}

	// HttpOnlyはfalse（JavaScriptからアクセス可能にする）
	if csrfCookie.HttpOnly {
		t.Error("HttpOnly属性がfalseではありません")
	}

	if csrfCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite属性が正しくありません: got %v want %v", csrfCookie.SameSite, http.SameSiteLaxMode)
	}

	if csrfCookie.MaxAge != 24*60*60 {
		t.Errorf("MaxAgeが正しくありません: got %v want %v", csrfCookie.MaxAge, 24*60*60)
	}
}

func TestCSRFMiddleware_XForwardedProto(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false, // configではfalseでもX-Forwarded-Protoでtrueになる
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	cookies := rr.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == middleware.CSRFCookieName {
			csrfCookie = cookie
			break
		}
	}

	if csrfCookie == nil {
		t.Fatal("CSRFクッキーが設定されていません")
	}

	// X-Forwarded-Protoがhttpsの場合、Secure属性がtrueになる
	if !csrfCookie.Secure {
		t.Error("X-Forwarded-Proto: httpsの場合、Secure属性がtrueになるべきです")
	}
}

func TestGetCSRFTokenFromContext(t *testing.T) {
	t.Parallel()

	t.Run("トークンが設定されている場合", func(t *testing.T) {
		t.Parallel()
		token := "test_token_12345"
		ctx := middleware.SetCSRFTokenToContext(context.Background(), token)

		result := middleware.GetCSRFTokenFromContext(ctx)
		if result != token {
			t.Errorf("トークンが正しく取得できませんでした: got %v want %v", result, token)
		}
	})

	t.Run("トークンが設定されていない場合", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		result := middleware.GetCSRFTokenFromContext(ctx)
		if result != "" {
			t.Errorf("トークンが空文字列ではありません: got %v want %v", result, "")
		}
	})
}

func TestCSRFMiddleware_POST_EmptyFormToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// 空のCSRFトークンをフォームに含める
	form := url.Values{}
	form.Set("csrf_token", "")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: "valid_token",
	})

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("空のフォームトークンで403が返されませんでした: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_SkipPaths(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// CSRFトークンなしでDELETE /user_sessionにリクエスト
	req := httptest.NewRequest("DELETE", "/user_session", nil)
	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("スキップパスで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}

	if !handlerCalled {
		t.Error("スキップパスでハンドラーが呼び出されませんでした")
	}
}

func TestCSRFMiddleware_SkipPaths_POST(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// CSRFトークンなしでPOST /user_sessionにリクエスト
	req := httptest.NewRequest("POST", "/user_session", nil)
	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("スキップパスで200が返されませんでした: got %v want %v", rr.Code, http.StatusOK)
	}

	if !handlerCalled {
		t.Error("スキップパスでハンドラーが呼び出されませんでした")
	}
}

func TestCSRFMiddleware_NonSkipPath_StillProtected(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// スキップパス以外はCSRFトークンなしで403になることを確認
	req := httptest.NewRequest("DELETE", "/other_path", nil)
	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("スキップパス以外で403が返されませんでした: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_POST_EmptyCookieToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	csrfMiddleware := middleware.NewCSRF(cfg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	form := url.Values{}
	form.Set("csrf_token", "valid_token")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  middleware.CSRFCookieName,
		Value: "", // 空のクッキートークン
	})

	rr := httptest.NewRecorder()

	csrfMiddleware.Middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("空のクッキートークンで403が返されませんでした: got %v want %v", rr.Code, http.StatusForbidden)
	}
}
