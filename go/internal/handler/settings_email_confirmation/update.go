package settings_email_confirmation

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Updateは確認コードを検証し、メールアドレスを変更します (PATCH /settings/email/confirmation)。
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	emailConfirmationID := h.sessionMgr.GetSettingsEmailConfirmationID(r)
	if emailConfirmationID == "" {
		http.Redirect(w, r, string(templates.SettingsEmailPath()), http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	_, err := h.updateEmailUC.Execute(ctx, usecase.UpdateEmailInput{
		UserID:              user.ID,
		EmailConfirmationID: emailConfirmationID,
		Code:                r.FormValue("code"),
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			h.render(w, r, user, http.StatusUnprocessableEntity, ve)
			return
		}
		if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeResourceNotFound {
			httperror.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "メールアドレスの変更に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.sessionMgr.DeleteSettingsEmailConfirmationCookie(w)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_email_updated"))
	http.Redirect(w, r, string(templates.SettingsEmailPath()), http.StatusSeeOther)
}
