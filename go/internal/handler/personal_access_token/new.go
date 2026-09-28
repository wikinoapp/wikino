package personal_access_token

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	patpages "github.com/wikinoapp/wikino/go/internal/templates/pages/personal_access_token"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Newは個人アクセストークンの発行フォームを表示します (GET /s/{space_identifier}/settings/personal_access_tokens/new)。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.getPersonalAccessTokenNewUC.Execute(ctx, usecase.GetPersonalAccessTokenNewInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "個人アクセストークンの発行フォームのデータの取得に失敗")
		return
	}

	h.renderNewForm(w, r, user, http.StatusOK, patpages.NewData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     viewmodel.NewSpace(output.Space),
	})
}

// renderNewFormは発行フォームをstatusで描画します。フォームの表示と、拒否された送信の再表示の
// 両方がここを通ります。ステータスを書いた後のヘッダーは送られないため、Cache-Controlを
// 設定してからステータスを書きます。
func (h *Handler) renderNewForm(w http.ResponseWriter, r *http.Request, user *model.User, status int, data patpages.NewData) {
	ctx := r.Context()

	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label: i18n.T(ctx, "personal_access_token_new_breadcrumb_index"),
			Path:  templates.SpaceSettingsPersonalAccessTokensPath(data.Space.Identifier),
		},
		// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
		{
			Label:     i18n.T(ctx, "personal_access_token_new_breadcrumb"),
			IsCurrent: true,
		},
	}

	w.Header().Set("Cache-Control", "private, no-cache")
	w.WriteHeader(status)
	h.render(w, r, user, data.Space, templates.PageNamePersonalAccessTokenNew, "personal_access_token_new_title", "personal_access_token_new_breadcrumb_settings", trailingBreadcrumbs, patpages.New(data))
}
