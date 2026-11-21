package robots_txt

import (
	"io"
	"net/http"
)

// bodyはすべてのクローラーにすべてのパスを許可するrobots.txtの本文。
// 空の `Disallow:` で全許可を明示する
const body = "User-agent: *\nDisallow:\n"

// Showはrobots.txtを返します (GET /robots.txt)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.WriteString(w, body)
}
