package export_download

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Show sends the viewer to the archive of an export
// (GET /s/{space_identifier}/settings/exports/{export_id}/download).
//
// The response is a redirect to a URL signed for the object storage rather than the bytes
// themselves, so a large archive is fetched straight from the bucket instead of being streamed
// through this server.
//
// [Ja] Show は閲覧者をエクスポートのアーカイブへ送ります
// (GET /s/{space_identifier}/settings/exports/{export_id}/download)。
//
// 返すのはバイト列そのものではなく、オブジェクトストレージ向けに署名した URL へのリダイレクトである。
// 大きなアーカイブが本サーバーを経由してストリーミングされるのではなく、バケットから直接取得される
// ようにするためである。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	output, err := h.getExportDownloadUC.Execute(ctx, usecase.GetExportDownloadInput{
		SpaceIdentifier: model.SpaceIdentifier(chi.URLParam(r, "space_identifier")),
		ExportID:        model.ExportID(chi.URLParam(r, "export_id")),
		UserID:          user.ID,
	})
	if err != nil {
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

		slog.ErrorContext(ctx, "エクスポートのダウンロードに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, output.URL, http.StatusSeeOther)
}
