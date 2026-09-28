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

// IndexはスペースのOAuthアプリの一覧を表示します (GET /s/{space_identifier}/settings/oauth_applications)。
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.getOAuthApplicationsUC.Execute(ctx, usecase.GetOAuthApplicationsInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "OAuthアプリの一覧のデータの取得に失敗")
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)

	content := oauthapppages.Index(oauthapppages.IndexData{
		Space:        spaceVM,
		Applications: viewmodel.NewOAuthApplications(output.Applications, output.Creators),
		CanCreate:    output.CanCreate,
	})

	// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label:     i18n.T(ctx, "oauth_application_index_breadcrumb"),
			IsCurrent: true,
		},
	}

	w.Header().Set("Cache-Control", "private, no-cache")
	h.render(w, r, user, spaceVM, templates.PageNameOAuthApplicationIndex, "oauth_application_index_title", nil, "oauth_application_index_breadcrumb_settings", trailingBreadcrumbs, content)
}
