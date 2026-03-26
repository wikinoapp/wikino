package middleware

import (
	"net/http"
	"time"

	"github.com/wikinoapp/wikino/go/internal/timezone"
)

const (
	// timeZoneCookieName はブラウザのタイムゾーンを保存するクッキー名
	timeZoneCookieName = "wikino_time_zone"
)

// TimeZone はリクエストからタイムゾーンを解決しコンテキストに格納するミドルウェア
// 認証ミドルウェアより後に配置すること（UserFromContextを使用するため）
func TimeZone(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tz := resolveTimeZone(r)
		ctx := timezone.ToContext(r.Context(), tz)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// resolveTimeZone はリクエストからタイムゾーンを解決する
// 優先順位: 1. ログインユーザーの設定 → 2. クッキー → 3. UTC
func resolveTimeZone(r *http.Request) string {
	if user := UserFromContext(r.Context()); user != nil && user.TimeZone != "" {
		return user.TimeZone
	}

	if c, err := r.Cookie(timeZoneCookieName); err == nil && c.Value != "" {
		if _, err := time.LoadLocation(c.Value); err == nil {
			return c.Value
		}
	}

	return "UTC"
}
