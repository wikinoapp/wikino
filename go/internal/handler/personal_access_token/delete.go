package personal_access_token

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

// Deleteは自分の個人アクセストークンを失効します (DELETE /s/{space_identifier}/settings/personal_access_tokens/{personal_access_token_id})。
//
// トークンの行は消さずに失効日時を記録する。失効したトークンは一覧から消え、次のAPI
// リクエストから受け付けられなくなる。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.revokePersonalAccessTokenUC.Execute(ctx, usecase.RevokePersonalAccessTokenInput{
		SpaceIdentifier:       spaceIdentifier,
		UserID:                user.ID,
		PersonalAccessTokenID: model.PersonalAccessTokenID(chi.URLParam(r, "personal_access_token_id")),
	})
	if err != nil {
		h.handleError(w, r, err, "個人アクセストークンの失効に失敗")
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_personal_access_token_revoked", map[string]any{"Name": output.Token.Name}))
	http.Redirect(w, r, string(templates.SpaceSettingsPersonalAccessTokensPath(viewmodel.NewSpaceIdentifier(output.Space.Identifier))), http.StatusSeeOther)
}
