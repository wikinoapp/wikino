// Package personal_access_tokenは個人アクセストークンの画面のHTTPハンドラーを提供します。
package personal_access_token

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Handlerは個人アクセストークンの一覧・発行・失効の画面を提供します。
type Handler struct {
	cfg                         *config.Config
	flashMgr                    *session.FlashManager
	getPersonalAccessTokensUC   *usecase.GetPersonalAccessTokensUsecase
	getPersonalAccessTokenNewUC *usecase.GetPersonalAccessTokenNewUsecase
	createPersonalAccessTokenUC *usecase.CreatePersonalAccessTokenUsecase
	revokePersonalAccessTokenUC *usecase.RevokePersonalAccessTokenUsecase
}

// NewHandlerは個人アクセストークンのハンドラーを生成します。
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getPersonalAccessTokensUC *usecase.GetPersonalAccessTokensUsecase,
	getPersonalAccessTokenNewUC *usecase.GetPersonalAccessTokenNewUsecase,
	createPersonalAccessTokenUC *usecase.CreatePersonalAccessTokenUsecase,
	revokePersonalAccessTokenUC *usecase.RevokePersonalAccessTokenUsecase,
) *Handler {
	return &Handler{
		cfg:                         cfg,
		flashMgr:                    flashMgr,
		getPersonalAccessTokensUC:   getPersonalAccessTokensUC,
		getPersonalAccessTokenNewUC: getPersonalAccessTokenNewUC,
		createPersonalAccessTokenUC: createPersonalAccessTokenUC,
		revokePersonalAccessTokenUC: revokePersonalAccessTokenUC,
	}
}

// renderは個人アクセストークンの画面を、それらが共有するレイアウトへ描画する。どの画面も
// スペース設定の下にあり、違うのはページ名・タイトル・パンくずの末尾・中身だけなので、ヘッダーと
// スペース設定までの経路はここで一度だけ決める。
//
// 各画面が持つ文言はpageNameから導かず呼び出し元が名指しする。画面が持つキーをgrepで追える
// ようにするためである。
func (h *Handler) render(
	w http.ResponseWriter,
	r *http.Request,
	user *model.User,
	space viewmodel.Space,
	pageName templates.PageName,
	titleKey string,
	breadcrumbSettingsKey string,
	trailingBreadcrumbs []components.BreadcrumbItem,
	content templ.Component,
) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, titleKey, map[string]any{"SpaceName": space.Name})
	meta.CurrentSpaceIdentifier = space.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: pageName,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: space.Identifier,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: space.Name,
					Path:  templates.SpacePath(space.Identifier),
				},
				components.BreadcrumbItem{
					Label: i18n.T(ctx, breadcrumbSettingsKey),
					Path:  templates.SpaceSettingsPath(space.Identifier),
				},
			), trailingBreadcrumbs...),
		},
	}

	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleErrorはユースケースから返ったエラーを、それに応じたレスポンスへ変える。閲覧者が
// 開けない画面は「見つからない」として答え、そのスペースで誰が何をできるかを外部から試して
// 確かめられないようにする。
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
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

	slog.ErrorContext(ctx, logMsg, "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
