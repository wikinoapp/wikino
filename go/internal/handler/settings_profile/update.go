package settings_profile

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	settingspages "github.com/wikinoapp/wikino/go/internal/templates/pages/settings"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Updateはプロフィールを保存します (PATCH /settings/profile)。
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	input := usecase.UpdateProfileInput{
		UserID:      user.ID,
		Atname:      r.FormValue("atname"),
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}

	if _, err := h.updateProfileUC.Execute(ctx, input); err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			h.render(w, r, user, settingspages.ProfileData{
				CSRFToken:   middleware.GetCSRFTokenFromContext(ctx),
				FormErrors:  ve,
				Atname:      input.Atname,
				Name:        input.Name,
				Description: input.Description,
			})
			return
		}
		if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeResourceNotFound {
			httperror.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "プロフィールの更新に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_profile_updated"))
	http.Redirect(w, r, string(templates.SettingsProfilePath()), http.StatusSeeOther)
}
