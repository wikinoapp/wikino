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

// NewはOAuthアプリの登録フォームを表示します (GET /s/{space_identifier}/settings/oauth_applications/new)。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.getOAuthApplicationNewUC.Execute(ctx, usecase.GetOAuthApplicationNewInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "OAuthアプリの登録フォームのデータの取得に失敗")
		return
	}

	h.renderNewForm(w, r, user, http.StatusOK, oauthapppages.NewData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     viewmodel.NewSpace(output.Space),
	})
}

// renderNewFormは登録フォームをstatusで描画します。フォームの表示と、拒否された送信の再表示の
// 両方がここを通ります。ステータスを書いた後のヘッダーは送られないため、Cache-Controlを
// 設定してからステータスを書きます。
func (h *Handler) renderNewForm(w http.ResponseWriter, r *http.Request, user *model.User, status int, data oauthapppages.NewData) {
	ctx := r.Context()

	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label: i18n.T(ctx, "oauth_application_new_breadcrumb_index"),
			Path:  templates.SpaceSettingsOAuthApplicationsPath(data.Space.Identifier),
		},
		// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
		{
			Label:     i18n.T(ctx, "oauth_application_new_breadcrumb"),
			IsCurrent: true,
		},
	}

	w.Header().Set("Cache-Control", "private, no-cache")
	w.WriteHeader(status)
	h.render(w, r, user, data.Space, templates.PageNameOAuthApplicationNew, "oauth_application_new_title", nil, "oauth_application_new_breadcrumb_settings", trailingBreadcrumbs, oauthapppages.New(data))
}
