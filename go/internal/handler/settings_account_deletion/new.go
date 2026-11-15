package settings_account_deletion

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
)

// Newはアカウントの削除の確認画面を表示します (GET /settings/account/deletion/new)。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	h.renderForm(w, r, user, http.StatusOK, "", nil)
}
