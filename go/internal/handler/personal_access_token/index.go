package personal_access_token

import (
	"net/http"
	"time"

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

// Indexは自分の個人アクセストークンの一覧を表示します (GET /s/{space_identifier}/settings/personal_access_tokens)。
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.getPersonalAccessTokensUC.Execute(ctx, usecase.GetPersonalAccessTokensInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "個人アクセストークンの一覧のデータの取得に失敗")
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)

	content := patpages.Index(patpages.IndexData{
		CSRFToken: middleware.GetCSRFTokenFromContext(ctx),
		Space:     spaceVM,
		Tokens:    viewmodel.NewPersonalAccessTokens(output.Tokens, time.Now()),
		CanCreate: output.CanCreate,
		CanDelete: output.CanDelete,
	})

	// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label:     i18n.T(ctx, "personal_access_token_index_breadcrumb"),
			IsCurrent: true,
		},
	}

	w.Header().Set("Cache-Control", "private, no-cache")
	h.render(w, r, user, spaceVM, templates.PageNamePersonalAccessTokenIndex, "personal_access_token_index_title", "personal_access_token_index_breadcrumb_settings", trailingBreadcrumbs, content)
}
