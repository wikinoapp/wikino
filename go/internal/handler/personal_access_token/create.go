package personal_access_token

import (
	"log/slog"
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

// Createは個人アクセストークンを発行します (POST /s/{space_identifier}/settings/personal_access_tokens)。
//
// 発行に成功したら、リダイレクトせずにその応答でトークンの値を表示する。データベースには
// ダイジェストしか残らないため、値を次のリクエストへ持ち越すにはCookieなどのクライアント側に
// 値を置くことになる。値がブラウザに渡る経路をこの応答の本文1つに限り、キャッシュにも残さない。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	input := usecase.CreatePersonalAccessTokenInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
		Name:            r.PostFormValue("name"),
		Scopes:          r.PostForm["scopes"],
		ExpirationDays:  r.PostFormValue("expiration_days"),
	}

	output, err := h.createPersonalAccessTokenUC.Execute(ctx, input)
	if err != nil {
		h.handleCreateError(w, r, err, user, input)
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)

	content := patpages.Create(patpages.CreateData{
		Space:      spaceVM,
		Token:      viewmodel.NewPersonalAccessToken(output.Token, time.Now()),
		TokenValue: output.TokenValue,
	})

	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label: i18n.T(ctx, "personal_access_token_create_breadcrumb_index"),
			Path:  templates.SpaceSettingsPersonalAccessTokensPath(spaceVM.Identifier),
		},
		{
			Label:     i18n.T(ctx, "personal_access_token_create_breadcrumb"),
			IsCurrent: true,
		},
	}

	w.Header().Set("Cache-Control", "no-store")
	h.render(w, r, user, spaceVM, templates.PageNamePersonalAccessTokenCreate, "personal_access_token_create_title", "personal_access_token_create_breadcrumb_settings", trailingBreadcrumbs, content)
}

// handleCreateErrorは入力が拒否されたときに送信された内容を戻したフォームを再描画し、
// それ以外は他の画面と同じ形で応答する。
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, input usecase.CreatePersonalAccessTokenInput) {
	ctx := r.Context()

	ve := model.AsValidationError(err)
	if ve == nil {
		h.handleError(w, r, err, "個人アクセストークンの発行に失敗")
		return
	}

	output, getErr := h.getPersonalAccessTokenNewUC.Execute(ctx, usecase.GetPersonalAccessTokenNewInput{
		SpaceIdentifier: input.SpaceIdentifier,
		UserID:          user.ID,
	})
	if getErr != nil {
		h.handleError(w, r, getErr, "フォーム再表示用データの取得に失敗")
		return
	}

	h.renderNewForm(w, r, user, http.StatusUnprocessableEntity, patpages.NewData{
		CSRFToken:      middleware.GetCSRFTokenFromContext(ctx),
		Space:          viewmodel.NewSpace(output.Space),
		FormErrors:     ve,
		Name:           input.Name,
		Scopes:         input.Scopes,
		ExpirationDays: input.ExpirationDays,
	})
}
