package oauth_grant

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	oauthgrantpages "github.com/wikinoapp/wikino/go/internal/templates/pages/oauth_grant"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Indexは自分が許可した連携中のアプリの一覧を表示します (GET /s/{space_identifier}/settings/oauth_grants)。
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.getOAuthGrantsUC.Execute(ctx, usecase.GetOAuthGrantsInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "連携中のアプリの一覧のデータの取得に失敗")
		return
	}

	space := viewmodel.NewSpace(output.Space)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "oauth_grant_index_title", map[string]any{"SpaceName": space.Name})
	meta.CurrentSpaceIdentifier = space.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameOAuthGrantIndex,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: space.Identifier,
		},

		// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: space.Name,
					Path:  templates.SpacePath(space.Identifier),
				},
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "oauth_grant_index_breadcrumb_settings"),
					Path:  templates.SpaceSettingsPath(space.Identifier),
				},
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "oauth_grant_index_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	content := oauthgrantpages.Index(oauthgrantpages.IndexData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     space,
		Grants:    viewmodel.NewOAuthGrants(output.Grants, output.Applications),
		CanDelete: output.CanDelete,
	})

	w.Header().Set("Cache-Control", "private, no-cache")
	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
