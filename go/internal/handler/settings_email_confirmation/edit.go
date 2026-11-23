package settings_email_confirmation

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates"
)

// Editは確認コードを入力する画面を表示します (GET /settings/email/confirmation/edit)。
// 確認を始めていない (確認のIDのCookieが無い) ときは、メールアドレスの変更の画面へ送ります。
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	if h.sessionMgr.GetSettingsEmailConfirmationID(r) == "" {
		http.Redirect(w, r, string(templates.SettingsEmailPath()), http.StatusFound)
		return
	}

	h.render(w, r, user, http.StatusOK, nil)
}
