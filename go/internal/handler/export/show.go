package export

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
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

	h.render(w, r, user, spaceVM, templates.PageNameExportShow, "export_show_title", "export_show_breadcrumb_settings", content)
}
