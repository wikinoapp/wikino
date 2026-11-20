// Package space_settings_generalはスペースの一般設定のHTTPハンドラーを提供します。
package space_settings_general

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

// Handlerはスペースの一般設定を編集する画面と、その送信先の保存処理を提供します。
type Handler struct {
	cfg                       *config.Config
	flashMgr                  *session.FlashManager
	getSpaceSettingsGeneralUC *usecase.GetSpaceSettingsGeneralUsecase
	updateSpaceUC             *usecase.UpdateSpaceUsecase
}

// NewHandlerはスペースの一般設定のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSpaceSettingsGeneralUC *usecase.GetSpaceSettingsGeneralUsecase,
	updateSpaceUC *usecase.UpdateSpaceUsecase,
) *Handler {
	return &Handler{
		cfg:                       cfg,
		flashMgr:                  flashMgr,
		getSpaceSettingsGeneralUC: getSpaceSettingsGeneralUC,
		updateSpaceUC:             updateSpaceUC,
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

// renderは一般設定画面を、スペースの他の画面と共有するレイアウトへ描画します。パンくずは
// スペース設定のトップを通り、この画面自身の非リンクな現在項目で締めます。
func (h *Handler) render(w http.ResponseWriter, r *http.Request, user *model.User, data spacepages.SettingsGeneralData) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "space_settings_general_title", map[string]any{"SpaceName": data.Space.Name})
	meta.CurrentSpaceIdentifier = data.Space.Identifier

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSpaceSettingsGeneral,
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
					Label: i18n.T(ctx, "space_settings_general_breadcrumb_settings"),
					Path:  templates.SpaceSettingsPath(data.Space.Identifier),
				},
				// この画面が現在地のため、経路はaria-currentを持つリンク無しの項目で締める。
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "space_settings_general_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	if err := layouts.Default(layoutData, spacepages.SettingsGeneral(data)).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleErrorはユースケースから返ったエラーを、それに応じたレスポンスへ変えます。
// 閲覧者が変更できないスペースは「見つからない」として答え、そのスペースで誰が何を変更できるかを
// 外部から試して確かめられないようにします。
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
