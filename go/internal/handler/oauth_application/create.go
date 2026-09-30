package oauth_application

import (
	"log/slog"
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

// CreateはOAuthアプリを登録します (POST /s/{space_identifier}/settings/oauth_applications)。
//
// confidentialクライアントは、リダイレクトせずにその応答でクライアントシークレットを表示する。
// データベースにはダイジェストしか残らないため、値を次のリクエストへ持ち越すにはCookieなどの
// クライアント側に値を置くことになる。値がブラウザに渡る経路をこの応答の本文1つに限り、
// キャッシュにも残さない (個人アクセストークンの発行と同じ)。
//
// publicクライアントは示す秘密の値が無いため、詳細の画面へ303でリダイレクトする。再読み込みで
// 送信し直して2つ目のアプリを登録してしまうことを避けるためである。
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

	input := usecase.CreateOAuthApplicationInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
		Name:            r.PostFormValue("name"),
		RedirectURIs:    r.PostFormValue("redirect_uris"),
		ClientType:      r.PostFormValue("client_type"),
	}

	output, err := h.createOAuthApplicationUC.Execute(ctx, input)
	if err != nil {
		h.handleCreateError(w, r, err, user, input)
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)
	appVM := viewmodel.NewOAuthApplication(output.Application, nil)

	if output.ClientSecret == "" {
		h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_oauth_application_created", map[string]any{"Name": appVM.Name}))
		http.Redirect(w, r, string(templates.SpaceSettingsOAuthApplicationPath(spaceVM.Identifier, appVM.ID)), http.StatusSeeOther)
		return
	}

	content := oauthapppages.Create(oauthapppages.CreateData{
		Space:        spaceVM,
		Application:  appVM,
		ClientSecret: output.ClientSecret,
	})

	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label: i18n.T(ctx, "oauth_application_create_breadcrumb_index"),
			Path:  templates.SpaceSettingsOAuthApplicationsPath(spaceVM.Identifier),
		},
		{
			Label:     i18n.T(ctx, "oauth_application_create_breadcrumb"),
			IsCurrent: true,
		},
	}

	w.Header().Set("Cache-Control", "no-store")
	h.render(w, r, user, spaceVM, templates.PageNameOAuthApplicationCreate, "oauth_application_create_title", nil, "oauth_application_create_breadcrumb_settings", trailingBreadcrumbs, content)
}

// handleCreateErrorは入力が拒否されたときに送信された内容を戻したフォームを再描画し、
// それ以外は他の画面と同じ形で応答する。
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, input usecase.CreateOAuthApplicationInput) {
	ctx := r.Context()

	ve := model.AsValidationError(err)
	if ve == nil {
		h.handleError(w, r, err, "OAuthアプリの登録に失敗")
		return
	}

	output, getErr := h.getOAuthApplicationNewUC.Execute(ctx, usecase.GetOAuthApplicationNewInput{
		SpaceIdentifier: input.SpaceIdentifier,
		UserID:          user.ID,
	})
	if getErr != nil {
		h.handleError(w, r, getErr, "フォーム再表示用データの取得に失敗")
		return
	}

	h.renderNewForm(w, r, user, http.StatusUnprocessableEntity, oauthapppages.NewData{
		CSRFToken:    middleware.GetCSRFTokenFromContext(ctx),
		Space:        viewmodel.NewSpace(output.Space),
		FormErrors:   ve,
		Name:         input.Name,
		RedirectURIs: input.RedirectURIs,
		ClientType:   input.ClientType,
	})
}
