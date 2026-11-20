package settings_profile

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	settingspages "github.com/wikinoapp/wikino/go/internal/templates/pages/settings"
)

// Showはプロフィールの編集画面を表示します (GET /settings/profile)。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	h.render(w, r, user, http.StatusOK, settingspages.ProfileData{
		CSRFToken:   middleware.GetCSRFTokenFromContext(ctx),
		Atname:      user.Atname,
		Name:        user.Name,
		Description: user.Description,
	})
}
