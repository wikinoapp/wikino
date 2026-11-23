package apple_touch_icon

import "net/http"

// ShowはApple Touch Iconを返します (GET /apple-touch-icon.png・GET /apple-touch-icon-precomposed.png)。
// iOSはHTMLの指定に関係なくルート直下を取りに来るため、`/static` と同じファイルを2つのパスで返す
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, h.staticFS, "images/apple-touch-icon.png")
}
