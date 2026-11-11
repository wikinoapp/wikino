package middleware

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/wikinoapp/wikino/go/internal/darkmode"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// DarkModeは、ダークモードのフィーチャーフラグが閲覧者に有効かを判定する関数をコンテキストに載せるミドルウェア。
// 判定 (1クエリ) は、ページ全体を描画するときに初めて1回だけ行う。
// htmxのフラグメント・API・静的ファイルのリクエストは判定を呼ばないため、クエリを発行しない
func DarkMode(checker featureFlagChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// APIリファレンスは共通のHeadを使わずライトで表示するため、フラグの判定を省く
			if r.URL.Path == model.APIReferencePath {
				next.ServeHTTP(w, r)
				return
			}

			isEnabled := sync.OnceValue(func() bool {
				return isDarkModeEnabled(r, checker)
			})
			ctx := darkmode.ToContext(r.Context(), isEnabled)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isDarkModeEnabledは、device_tokenとログインセッションのCookieでダークモードのフラグを判定する。
// ログインしていない画面でもデバイス単位で効かせるため、両方のCookieを使う。
// 判定できないときは、ライトで表示するようfalseを返す
func isDarkModeEnabled(r *http.Request, checker featureFlagChecker) bool {
	deviceToken := ""
	if cookie, err := r.Cookie(DeviceTokenCookieName); err == nil {
		deviceToken = cookie.Value
	}

	sessionToken := ""
	if cookie, err := r.Cookie(session.CookieName); err == nil {
		sessionToken = cookie.Value
	}

	if deviceToken == "" && sessionToken == "" {
		return false
	}

	enabled, err := checker.IsEnabledForDevice(r.Context(), deviceToken, sessionToken, model.FeatureFlagDarkMode)
	if err != nil {
		slog.WarnContext(r.Context(), "ダークモードのフィーチャーフラグの判定でエラーが発生 (ライトで表示する)",
			"error", err,
			"path", r.URL.Path,
		)
		return false
	}

	return enabled
}
