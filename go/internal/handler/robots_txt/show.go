package robots_txt

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// bodyはすべてのクローラーにすべてのパスを許可するrobots.txtの本文。
// 空の `Disallow:` で全許可を明示する
const body = "User-agent: *\nDisallow:\n"

// etagはクローラーが取得済みの本文を再検証できるよう、本文から生成する
var etag = fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(body)))

// Showはrobots.txtを返します (GET /robots.txt・HEAD /robots.txt)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("ETag", etag)
	http.ServeContent(w, r, "robots.txt", time.Time{}, strings.NewReader(body))
}
