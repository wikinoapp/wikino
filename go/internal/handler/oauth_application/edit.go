package oauth_application

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	oauthapppages "github.com/wikinoapp/wikino/go/internal/templates/pages/oauth_application"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// EditはOAuthアプリの編集フォームを表示します (GET /s/{space_identifier}/settings/oauth_applications/{oauth_application_id}/edit)。
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	output, err := h.getOAuthApplicationEditUC.Execute(ctx, usecase.GetOAuthApplicationEditInput{
		SpaceIdentifier:    model.SpaceIdentifier(chi.URLParam(r, "space_identifier")),
		UserID:             user.ID,
		OAuthApplicationID: model.OAuthApplicationID(chi.URLParam(r, "oauth_application_id")),
	})
	if err != nil {
		h.handleError(w, r, err, "OAuthアプリの編集フォームのデータの取得に失敗")
		return
	}

	appVM := viewmodel.NewOAuthApplication(output.Application, nil)
	h.renderEditForm(w, r, user, http.StatusOK, oauthapppages.EditData{
		CSRFToken:   middleware.GetCSRFTokenFromContext(ctx),
		Space:       viewmodel.NewSpace(output.Space),
		Application: appVM,
		Version:     output.Application.Version,
		Fields: oauthapppages.FormFieldsData{
			Name:         appVM.Name,
			RedirectURIs: strings.Join(appVM.RedirectURIs, "\n"),
		},
	})
}

// renderEditFormは編集フォームをstatusで描画します。フォームの表示と、拒否された送信の再表示の
// 両方がここを通ります。ステータスを書いた後のヘッダーは送られないため、Cache-Controlを
// 設定してからステータスを書きます。
func (h *Handler) renderEditForm(w http.ResponseWriter, r *http.Request, user *model.User, status int, data oauthapppages.EditData) {
	ctx := r.Context()

	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label: i18n.T(ctx, "oauth_application_edit_breadcrumb_index"),
			Path:  templates.SpaceSettingsOAuthApplicationsPath(data.Space.Identifier),
		},
		{
			Label: data.Application.Name,
			Path:  templates.SpaceSettingsOAuthApplicationPath(data.Space.Identifier, data.Application.ID),
		},
		// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
		{
			Label:     i18n.T(ctx, "oauth_application_edit_breadcrumb"),
			IsCurrent: true,
		},
	}

	w.Header().Set("Cache-Control", "private, no-cache")
	w.WriteHeader(status)
	h.render(w, r, user, data.Space, templates.PageNameOAuthApplicationEdit, "oauth_application_edit_title", map[string]any{"Name": data.Application.Name}, "oauth_application_edit_breadcrumb_settings", trailingBreadcrumbs, oauthapppages.Edit(data))
}
