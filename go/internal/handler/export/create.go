package export

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create starts an export (POST /s/{space_identifier}/settings/exports).
//
// [Ja] Create はエクスポートを開始します (POST /s/{space_identifier}/settings/exports)。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	output, err := h.createExportUC.Execute(ctx, usecase.CreateExportInput{
		SpaceIdentifier: spaceIdentifier,
		UserID:          user.ID,
	})
	if err != nil {
		h.handleCreateError(w, r, err, spaceIdentifier)
		return
	}

	spaceIdentVM := viewmodel.NewSpaceIdentifier(output.Space.Identifier)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_export_started"))
	http.Redirect(w, r, string(templates.SpaceSettingsExportPath(spaceIdentVM, output.Export.ID.String())), http.StatusSeeOther)
}

// handleCreateError answers a start that was refused. A space that is already exporting sends the
// viewer to the export that holds it, which is the screen that says what is happening and is where
// the completion is followed. The screen hides the start button while that is the case, so this
// is reached when two requests raced.
//
// [Ja] handleCreateError は拒否された開始に応答する。既にエクスポート中のスペースでは、閲覧者を
// それを占めているエクスポートへ送る。その画面が何が起きているかを伝え、完了を追う場所でもある。
// その間は画面が開始ボタンを隠すため、ここに到達するのは 2 つのリクエストが競合したときである。
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, spaceIdentifier model.SpaceIdentifier) {
	if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeConflict {
		spaceIdentVM := viewmodel.NewSpaceIdentifier(spaceIdentifier)
		h.flashMgr.SetError(w, ae.UserMsg)

		destination := templates.NewSpaceSettingsExportPath(spaceIdentVM)
		if exportID := ae.Metadata["export_id"]; exportID != "" {
			destination = templates.SpaceSettingsExportPath(spaceIdentVM, exportID)
		}
		http.Redirect(w, r, string(destination), http.StatusSeeOther)
		return
	}

	h.handleError(w, r, err, "エクスポートの開始に失敗")
}
