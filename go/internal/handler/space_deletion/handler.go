// Package space_deletionはスペースの削除の確認画面と削除処理のHTTPハンドラーを提供します。
package space_deletion

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Handlerはスペースの削除の確認画面と、その送信先の削除処理を提供します。
type Handler struct {
	cfg                   *config.Config
	flashMgr              *session.FlashManager
	getSpaceDeletionNewUC *usecase.GetSpaceDeletionNewUsecase
	discardSpaceUC        *usecase.DiscardSpaceUsecase
}

// NewHandlerはスペースの削除のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSpaceDeletionNewUC *usecase.GetSpaceDeletionNewUsecase,
	discardSpaceUC *usecase.DiscardSpaceUsecase,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		flashMgr:              flashMgr,
		getSpaceDeletionNewUC: getSpaceDeletionNewUC,
		discardSpaceUC:        discardSpaceUC,
	}
}

// requestTargetはURLが指すスペースの識別子とログイン中のユーザーを読み、読めたかどうかを返します。
func requestTarget(w http.ResponseWriter, r *http.Request) (model.SpaceIdentifier, *model.User, bool) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return "", nil, false
	}

	return model.SpaceIdentifier(chi.URLParam(r, "space_identifier")), user, true
}

// renderは確認画面を、スペースの他の画面と共有するレイアウトへ描画します。パンくずは
// スペース設定のトップを通り、この画面自身の非リンクな現在項目で締めます。
func (h *Handler) render(w http.ResponseWriter, r *http.Request, user *model.User, status int, data spacepages.DeletionNewData) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "space_deletion_new_title", map[string]any{"SpaceName": data.Space.Name})
	meta.CurrentSpaceIdentifier = data.Space.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSpaceDeletionNew,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: data.Space.Identifier,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: data.Space.Name,
					Path:  templates.SpacePath(data.Space.Identifier),
				},
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "space_deletion_new_breadcrumb_settings"),
					Path:  templates.SpaceSettingsPath(data.Space.Identifier),
				},
				// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "space_deletion_new_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	// ログインした閲覧者だけが開く設定の画面のため、検索エンジンに載せない。
	w.Header().Set("X-Robots-Tag", "noindex")
	w.WriteHeader(status)
	if err := layouts.Default(layoutData, spacepages.DeletionNew(data)).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleErrorはユースケースから返ったエラーを、それに応じたレスポンスへ変えます。
// スペースを削除できない閲覧者には「見つからない」として答え、そのスペースで誰が何を変更できるかを
// 外部から試して確かめられないようにします。Rails版も同じく404を返していました。
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
