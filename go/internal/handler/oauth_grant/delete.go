package oauth_grant

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

// Deleteは自分が許可した連携を解除します (DELETE /s/{space_identifier}/settings/oauth_grants/{oauth_grant_id})。
//
// 許可の行は消さずに失効日時を記録し、許可に属するアクセストークン・リフレッシュトークンも
// すべて失効する。解除した連携は一覧から消え、アプリは次のAPIリクエストから受け付けられなくなる。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.revokeOAuthGrantUC.Execute(ctx, usecase.RevokeOAuthGrantInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
		OAuthGrantID:    model.OAuthGrantID(chi.URLParam(r, "oauth_grant_id")),
	})
	if err != nil {
		h.handleError(w, r, err, "連携の解除に失敗")
		return
	}

	if output.Application != nil {
		h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_oauth_grant_revoked", map[string]any{"Name": output.Application.Name}))
	}
	http.Redirect(w, r, string(templates.SpaceSettingsOAuthGrantsPath(viewmodel.NewSpaceIdentifier(output.Space.Identifier))), http.StatusSeeOther)
}
