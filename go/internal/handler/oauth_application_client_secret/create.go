package oauth_application_client_secret

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
	clientsecretpages "github.com/wikinoapp/wikino/go/internal/templates/pages/oauth_application_client_secret"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// CreateはOAuthアプリのクライアントシークレットを再発行します
// (POST /s/{space_identifier}/settings/oauth_applications/{oauth_application_id}/client_secret)。
//
// 新しいシークレットは、リダイレクトせずにこの応答で表示する。データベースにはダイジェスト
// しか残らないため、値がブラウザに渡る経路をこの応答の本文1つに限り、キャッシュにも残さない
// (アプリの登録と同じ)。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	output, err := h.regenerateOAuthApplicationClientSecretUC.Execute(ctx, usecase.RegenerateOAuthApplicationClientSecretInput{
		SpaceIdentifier:    model.SpaceIdentifier(chi.URLParam(r, "space_identifier")),
		UserID:             user.ID,
		OAuthApplicationID: model.OAuthApplicationID(chi.URLParam(r, "oauth_application_id")),
	})
	if err != nil {
		h.handleError(w, r, err, "クライアントシークレットの再発行に失敗")
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)
	appVM := viewmodel.NewOAuthApplication(output.Application, nil)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "oauth_application_client_secret_create_title", map[string]any{
		"Name":      appVM.Name,
		"SpaceName": spaceVM.Name,
	})
	meta.CurrentSpaceIdentifier = spaceVM.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameOAuthApplicationClientSecretCreate,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: spaceVM.Identifier,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: spaceVM.Name,
					Path:  templates.SpacePath(spaceVM.Identifier),
				},
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "oauth_application_client_secret_create_breadcrumb_settings"),
					Path:  templates.SpaceSettingsPath(spaceVM.Identifier),
				},
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "oauth_application_client_secret_create_breadcrumb_index"),
					Path:  templates.SpaceSettingsOAuthApplicationsPath(spaceVM.Identifier),
				},
				components.BreadcrumbItem{
					Label: appVM.Name,
					Path:  templates.SpaceSettingsOAuthApplicationPath(spaceVM.Identifier, appVM.ID),
				},
				// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "oauth_application_client_secret_create_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	content := clientsecretpages.Create(clientsecretpages.CreateData{
		Space:        spaceVM,
		Application:  appVM,
		ClientSecret: output.ClientSecret,
	})

	w.Header().Set("Cache-Control", "no-store")
	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
