package space_settings

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Showはスペース設定のトップを表示します (GET /s/{space_identifier}/settings)。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.getSpaceSettingsUC.Execute(ctx, usecase.GetSpaceSettingsInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "space_settings_title", map[string]any{"SpaceName": spaceVM.Name})
	meta.CurrentSpaceIdentifier = spaceVM.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSpaceSettings,
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
				// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "space_settings_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	content := spacepages.Settings(spacepages.SettingsData{
		Space:                       spaceVM,
		CanUpdateSpace:              output.Items.CanUpdateSpace,
		CanShowPersonalAccessTokens: output.Items.CanShowPersonalAccessTokens,
		CanShowOAuthGrants:          output.Items.CanShowOAuthGrants,
		CanShowOAuthApplications:    output.Items.CanShowOAuthApplications,
	})

	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleErrorはユースケースから返ったエラーを、それに応じたレスポンスへ変えます。
// 閲覧者が開けないスペース設定は「見つからない」として答え、そのスペースで誰が何を変更できるかを
// 外部から試して確かめられないようにします。Rails版も同じく404を返していました。
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			httperror.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "スペース設定のデータの取得に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
