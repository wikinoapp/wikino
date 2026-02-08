package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/config"
)

// CSRFCookieName はCSRFトークンを保存するクッキー名
const CSRFCookieName = "wikino_csrf_token"

// csrfSkipPaths はCSRF検証をスキップするパス
// Rails版のページからのリクエストはRails独自のCSRFトークンを使用しているため、
// Go版のCSRF検証を適用できない
var csrfSkipPaths = []string{
	"/user_session", // Rails版からのログアウトリクエスト対応
}

// csrfTokenContextKey はコンテキストにCSRFトークンを保存するためのキー
type csrfTokenContextKey struct{}

// CSRF はCSRF保護のためのミドルウェアを提供する
type CSRF struct {
	cfg *config.Config
}

// NewCSRF は新しいCSRFミドルウェアを作成する
func NewCSRF(cfg *config.Config) *CSRF {
	return &CSRF{
		cfg: cfg,
	}
}

// Middleware はCSRF保護ミドルウェアを返す
// GET/HEAD/OPTIONSリクエストではCSRFトークンを生成してコンテキストに設定
// その他のメソッドではCSRFトークンを検証する
func (c *CSRF) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// GET/HEAD/OPTIONSリクエストはCSRFトークンを生成して設定
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			token, err := c.getOrCreateCSRFToken(w, r)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			ctx = SetCSRFTokenToContext(ctx, token)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// スキップパスに一致する場合はCSRF検証をスキップ
		for _, path := range csrfSkipPaths {
			if r.URL.Path == path {
				next.ServeHTTP(w, r)
				return
			}
		}

		// POST/PATCH/PUT/DELETEリクエストはCSRFトークンを検証
		cookieToken, err := r.Cookie(CSRFCookieName)
		if err != nil || cookieToken.Value == "" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// フォームまたはヘッダーからトークンを取得
		formToken := r.FormValue("csrf_token")
		if formToken == "" {
			formToken = r.Header.Get("X-CSRF-Token")
		}

		// トークンが一致しない場合は403エラー
		if formToken == "" || formToken != cookieToken.Value {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// 検証成功後もコンテキストにトークンを設定
		ctx = SetCSRFTokenToContext(ctx, cookieToken.Value)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getOrCreateCSRFToken は既存のCSRFトークンを取得するか、新しく生成する
func (c *CSRF) getOrCreateCSRFToken(w http.ResponseWriter, r *http.Request) (string, error) {
	// 既存のトークンがあれば返す
	cookie, err := r.Cookie(CSRFCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	// 新しいトークンを生成
	token, err := generateCSRFToken()
	if err != nil {
		return "", err
	}

	// クッキーを設定
	c.setCSRFCookie(w, r, token)
	return token, nil
}

// setCSRFCookie はCSRFトークンをクッキーに設定する
func (c *CSRF) setCSRFCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := c.cfg.SessionSecure
	// リバースプロキシ経由のHTTPS接続を検出
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		secure = true
	}

	cookie := &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		Domain:   c.cfg.CookieDomain,
		Secure:   secure,
		HttpOnly: false, // JavaScriptからアクセス可能にする（AJAXリクエスト用）
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60, // 24時間
	}
	http.SetCookie(w, cookie)
}

// SetCSRFTokenToContext はコンテキストにCSRFトークンを設定する
func SetCSRFTokenToContext(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfTokenContextKey{}, token)
}

// GetCSRFTokenFromContext はコンテキストからCSRFトークンを取得する
func GetCSRFTokenFromContext(ctx context.Context) string {
	token, ok := ctx.Value(csrfTokenContextKey{}).(string)
	if !ok {
		return ""
	}
	return token
}

// generateCSRFToken は安全なCSRFトークンを生成する
// 32バイトのランダムデータをBase64エンコードして44文字のトークンを生成
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
