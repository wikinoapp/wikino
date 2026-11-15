package settings_two_factor_auth_recovery_code

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
)

// Indexは残っているリカバリーコードを表示します (GET /settings/two_factor_auth/recovery_codes)。
// 二要素認証が有効でないときは、設定の画面へ送ります。
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	h.renderCodes(w, r, user, http.StatusOK, nil)
}
