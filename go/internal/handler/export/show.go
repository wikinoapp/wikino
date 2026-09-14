package export

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	exportpages "github.com/wikinoapp/wikino/go/internal/templates/pages/export"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show follows one export (GET /s/{space_identifier}/settings/exports/{export_id}).
//
// [Ja] Show はエクスポート 1 件の経過を表示します (GET /s/{space_identifier}/settings/exports/{export_id})。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	exportID := model.ExportID(chi.URLParam(r, "export_id"))

	output, err := h.getExportShowUC.Execute(ctx, usecase.GetExportShowInput{
		SpaceIdentifier: spaceIdentifier,
		ExportID:        exportID,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleError(w, r, err, "エクスポート状態表示画面のデータの取得に失敗")
		return
	}

	spaceVM := viewmodel.NewSpace(output.Space)

	content := exportpages.Show(exportpages.ShowData{
		Space:  spaceVM,
		Export: viewmodel.NewExport(output.Export, time.Now()),
	})

	// The trail continues through the screen an export is started from, so that the two export
	// screens do not end with the same item and the way back to the start screen is on the trail as
	// well as in the body (where it only appears once an export is over). The current item names the
	// export this screen follows, which is the time it was queued at: an export has no title, and
	// the space can hold more than one.
	//
	// [Ja] 経路はエクスポートを開始する画面を通して続ける。エクスポートの 2 画面が同じ項目で終わら
	// ないようにし、開始画面へ戻る導線を本文 (エクスポートが終わってから出る) だけでなく経路にも
	// 置くためである。現在地の項目はこの画面が追っているエクスポートを表し、それは投入された時刻に
	// なる。エクスポートはタイトルを持たず、スペースは複数のエクスポートを持ちうるためである。
	trailingBreadcrumbs := []components.BreadcrumbItem{
		{
			Label: i18n.T(ctx, "export_show_breadcrumb_exports"),
			Path:  templates.NewSpaceSettingsExportPath(spaceVM.Identifier),
		},
		{
			Label:     templates.FormatDateTime(ctx, output.Export.CreatedAt),
			IsCurrent: true,
		},
	}

	h.render(w, r, user, spaceVM, templates.PageNameExportShow, "export_show_title", "export_show_breadcrumb_settings", trailingBreadcrumbs, content)
}
