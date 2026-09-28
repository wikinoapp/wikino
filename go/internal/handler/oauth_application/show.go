package oauth_application

import (
	"net/http"

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

// ShowはOAuthアプリの詳細を表示します (GET /s/{space_identifier}/settings/oauth_applications/{oauth_application_id})。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.getOAuthApplicationUC.Execute(ctx, usecase.GetOAuthApplicationInput{
		SpaceIdentifier:    spaceIdentifier,
		UserID:             user.ID,
		OAuthApplicationID: model.OAuthApplicationID(chi.URLParam(r, "oauth_application_id")),
	})
	if err != nil {
		h.handleError(w, r, err, "OAuthアプリの詳細のデータの取得に失敗")
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)
	appVM := viewmodel.NewOAuthApplication(output.Application, output.Creators)

	content := oauthapppages.Show(oauthapppages.ShowData{
		CSRFToken:   middleware.GetCSRFTokenFromContext(ctx),
		Space:       spaceVM,
		Application: appVM,
		CanUpdate:   output.CanUpdate,
		CanDelete:   output.CanDelete,
	})

	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label: i18n.T(ctx, "oauth_application_show_breadcrumb_index"),
			Path:  templates.SpaceSettingsOAuthApplicationsPath(spaceVM.Identifier),
		},
		// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
		{
			Label:     appVM.Name,
			IsCurrent: true,
		},
	}

	w.Header().Set("Cache-Control", "private, no-cache")
	h.render(w, r, user, spaceVM, templates.PageNameOAuthApplicationShow, "oauth_application_show_title", map[string]any{"Name": appVM.Name}, "oauth_application_show_breadcrumb_settings", trailingBreadcrumbs, content)
}
