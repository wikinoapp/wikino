package export

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	exportpages "github.com/wikinoapp/wikino/go/internal/templates/pages/export"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Newはエクスポートを開始する画面を表示します (GET /s/{space_identifier}/settings/exports/new)。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.getExportNewUC.Execute(ctx, usecase.GetExportNewInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "エクスポート開始画面のデータの取得に失敗")
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)

	var inProgressVM *viewmodel.Export
	if output.InProgressExport != nil {
		vm := viewmodel.NewExport(output.InProgressExport, time.Now())
		inProgressVM = &vm
	}

	content := exportpages.New(exportpages.NewData{
		CSRFToken:        middleware.GetCSRFTokenFromContext(ctx),
		Space:            spaceVM,
		InProgressExport: inProgressVM,
	})

	// 経路はここで終わる。この画面がエクスポートを開始する場所であり、自身の名前が現在地の
	// 項目になる。
	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label:     i18n.T(ctx, "export_new_breadcrumb"),
			IsCurrent: true,
		},
	}

	h.render(w, r, user, spaceVM, templates.PageNameExportNew, "export_new_title", "export_new_breadcrumb_settings", trailingBreadcrumbs, content)
}

// renderはエクスポートの画面を、それらが共有するレイアウトへ描画する。どちらもスペースの設定
// の下にあり、違うのはページ名・文言・パンくずの末尾・中身だけなので、ヘッダーとスペース設定までの
// 経路とナビはここで一度だけ決める。
//
// 各画面が持つ文言はpageNameから導かず呼び出し元が名指しする。画面が持つキーをgrepで追えるよう
// にし、ページ名の変更でラベルがキー自体に化けないようにするためである。i18n.Tは見つからない
// メッセージにキーをそのまま返す。
//
// スペース設定より下の経路は、文言キーをもう1つ受け取るのではなく項目そのものを呼び出し元から
// 受け取る。2つの画面は末尾の文言だけでなく形が違うためである。開始画面は自身で終わり、詳細画面は
// 開始画面を通って、追っているエクスポートまで続く。その名前はエクスポート自身のもので、メッセージ
// から引くことはできない。
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
			Items: append([]components.BreadcrumbItem{
				{
					Path:      templates.HomePath(),
					IconName:  "house-regular",
					AriaLabel: i18n.T(ctx, "breadcrumb_home"),
				},
				{
					Label: space.Name,
					Path:  templates.SpacePath(space.Identifier),
				},
				{
					Label: i18n.T(ctx, breadcrumbSettingsKey),
					Path:  templates.SpaceSettingsPath(space.Identifier),
				},
			}, trailingBreadcrumbs...),
		},
	}

	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleErrorはエクスポートのユースケースから返ったエラーを、それに応じたレスポンスへ変える。
// 閲覧者がエクスポートできないスペースは「見つからない」として答える。Rails版も同じで、スペースが
// 存在するが手が届かないと部外者に伝えることは、何も伝えないより多くを語ってしまう。
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	ctx := r.Context()

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			handler.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, logMsg, "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
