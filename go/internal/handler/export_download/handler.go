// Package export_download provides the HTTP handler that hands out the archive of an export.
//
// [Ja] export_download パッケージはエクスポートのアーカイブを渡す HTTP ハンドラーを提供します。
package export_download

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler serves the download of an export.
//
// [Ja] Handler はエクスポートのダウンロードを提供します。
type Handler struct {
	getExportDownloadUC *usecase.GetExportDownloadUsecase
}

// NewHandler creates an export download Handler.
//
// [Ja] NewHandler はエクスポートダウンロードのハンドラーを生成します。
func NewHandler(getExportDownloadUC *usecase.GetExportDownloadUsecase) *Handler {
	return &Handler{getExportDownloadUC: getExportDownloadUC}
}
