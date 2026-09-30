package middleware

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/session"
)

// APIResponseHeaderは公開Web APIのレスポンスに共通のヘッダーを付けるミドルウェア。
// OAuthのトークン・失効エンドポイントでは、レート制限やボディサイズ制限の応答にもno-storeを付ける。
// メンテナンス中の503やパニック時の500にも付けるため、それらのミドルウェアより外側に置く。
// ハンドラーは必要に応じて値を上書きできる (ヘッダーはレスポンスを書き始めるまで変更できる)。
//
//   - X-Robots-Tag: APIのレスポンスを検索エンジンにインデックスさせない
//   - Content-Security-Policy・X-Frame-Options: ブラウザで開いたレスポンスを他のサイトに埋め込ませない
//   - Cache-Control: トークンで認証したレスポンスをキャッシュに残さない
//   - X-Content-Type-Options: 宣言したContent-Type以外の形式としてブラウザに解釈させない
func APIResponseHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if session.OAuthTokenPathPattern.MatchString(r.URL.Path) {
			w.Header().Set("Cache-Control", "no-store")
		}
		if session.IsAPIPath(r.URL.Path) {
			h := w.Header()
			h.Set("X-Robots-Tag", "noindex")
			h.Set("Content-Security-Policy", "frame-ancestors 'none'")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Cache-Control", "no-store")
			h.Set("X-Content-Type-Options", "nosniff")
		}
		next.ServeHTTP(w, r)
	})
}
