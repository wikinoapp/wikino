package favicon

import "net/http"

// Showはファビコンを返します (GET /favicon.ico)。
// ブラウザはHTMLの指定に関係なくルート直下を取りに来るため、`/static` と同じファイルをここでも返す
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, h.staticFS, "images/favicon.ico")
}
