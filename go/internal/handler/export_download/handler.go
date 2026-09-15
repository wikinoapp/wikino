// Package export_downloadはエクスポートのアーカイブを渡すHTTPハンドラーを提供します。
package export_download

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはエクスポートのダウンロードを提供します。
type Handler struct {
	getExportDownloadUC *usecase.GetExportDownloadUsecase
}

// NewHandlerはエクスポートダウンロードのハンドラーを生成します。
func NewHandler(getExportDownloadUC *usecase.GetExportDownloadUsecase) *Handler {
	return &Handler{getExportDownloadUC: getExportDownloadUC}
}
