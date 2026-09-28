package oauth_application

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// DeleteはOAuthアプリを削除します (DELETE /s/{space_identifier}/settings/oauth_applications/{oauth_application_id})。
//
// アプリの行は消さずに削除日時を記録し、そのアプリの許可とトークンをすべて失効する。削除した
// アプリは一覧から消え、連携しているクライアントは次のリクエストから使えなくなる。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	output, err := h.deleteOAuthApplicationUC.Execute(ctx, usecase.DeleteOAuthApplicationInput{
		SpaceIdentifier:    model.SpaceIdentifier(chi.URLParam(r, "space_identifier")),
		UserID:             user.ID,
		OAuthApplicationID: model.OAuthApplicationID(chi.URLParam(r, "oauth_application_id")),
	})
	if err != nil {
		h.handleError(w, r, err, "OAuthアプリの削除に失敗")
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_oauth_application_deleted", map[string]any{"Name": output.Application.Name}))
	http.Redirect(w, r, string(templates.SpaceSettingsOAuthApplicationsPath(viewmodel.NewSpaceIdentifier(output.Space.Identifier))), http.StatusSeeOther)
}
