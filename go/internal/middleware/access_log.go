package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewAccessLogは1リクエストにつき1行、メソッド・パス・ステータス・応答のバイト数・処理時間・
// 接続元のアドレスをloggerで出力するミドルウェアを返す。
//
// URLはクエリ文字列を除いたパスだけを出力する。認可要求の端末名 (`device_name`) や、ログインの
// 戻り先 (`back`) に入った認可URLなど、ログに残さない値がクエリ文字列で渡るため。
func NewAccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// ハンドラーがpanicしてもリクエストの記録を残すため、deferで出力する。
			defer func() {
				status := ww.Status()
				if status == 0 {
					status = http.StatusOK
				}
				logger.InfoContext(r.Context(), "HTTPリクエスト",
					"method", r.Method,
					"path", r.URL.EscapedPath(),
					"status", status,
					"bytes", ww.BytesWritten(),
					"duration", time.Since(startedAt),
					"remote_addr", r.RemoteAddr,
				)
			}()
			next.ServeHTTP(ww, r)
		})
	}
}
