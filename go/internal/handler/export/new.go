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

// New shows the screen an export is started from (GET /s/{space_identifier}/settings/exports/new).
//
// [Ja] New はエクスポートを開始する画面を表示します (GET /s/{space_identifier}/settings/exports/new)。
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

	// The trail ends here: this screen is where an export is started from, so its own name is the
	// current item.
	//
	// [Ja] 経路はここで終わる。この画面がエクスポートを開始する場所であり、自身の名前が現在地の
	// 項目になる。
	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label:     i18n.T(ctx, "export_new_breadcrumb"),
			IsCurrent: true,
		},
	}

	h.render(w, r, user, spaceVM, templates.PageNameExportNew, "export_new_title", "export_new_breadcrumb_settings", trailingBreadcrumbs, content)
}

// render draws one of the export screens into the layout they share. Both sit under the settings
// of a space and differ only in their page name, their wording, the tail of their breadcrumb and
// their content, so the header, the trail up to the space settings and the navigation are decided
// once here.
//
// The wording each screen owns is named by the caller rather than derived from pageName, so that a
// key the screens carry stays greppable and renaming a page name cannot silently turn a label into
// the key itself, which is what i18n.T returns for a message it cannot find.
//
// The trail below the space settings is supplied by the caller as whole items rather than as one
// more wording key: the two screens differ in the shape of their tail, not only in its wording. The
// start screen ends at itself, while the detail screen continues through the start screen to the
// export it follows, whose name is that export's own and cannot come from a message.
//
// [Ja] render はエクスポートの画面を、それらが共有するレイアウトへ描画する。どちらもスペースの設定
// の下にあり、違うのはページ名・文言・パンくずの末尾・中身だけなので、ヘッダーとスペース設定までの
// 経路とナビはここで一度だけ決める。
//
// 各画面が持つ文言は pageName から導かず呼び出し元が名指しする。画面が持つキーを grep で追えるよう
// にし、ページ名の変更でラベルがキー自体に化けないようにするためである。i18n.T は見つからない
// メッセージにキーをそのまま返す。
//
// スペース設定より下の経路は、文言キーをもう 1 つ受け取るのではなく項目そのものを呼び出し元から
// 受け取る。2 つの画面は末尾の文言だけでなく形が違うためである。開始画面は自身で終わり、詳細画面は
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

// handleError turns an error from an export usecase into the response it deserves. A space the
// viewer may not export is answered as not found, which is what the Rails version did as well:
// telling an outsider that the space exists but is out of reach says more than nothing does.
//
// [Ja] handleError はエクスポートのユースケースから返ったエラーを、それに応じたレスポンスへ変える。
// 閲覧者がエクスポートできないスペースは「見つからない」として答える。Rails 版も同じで、スペースが
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
