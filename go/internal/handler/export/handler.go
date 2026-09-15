// Package exportはスペースのエクスポート画面のHTTPハンドラーを提供します。
package export

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはエクスポートを開始する画面と、その経過を追う画面を提供します。
type Handler struct {
	cfg             *config.Config
	flashMgr        *session.FlashManager
	getExportNewUC  *usecase.GetExportNewUsecase
	getExportShowUC *usecase.GetExportShowUsecase
	createExportUC  *usecase.CreateExportUsecase
}

// NewHandlerはエクスポートのハンドラーを生成します。
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getExportNewUC *usecase.GetExportNewUsecase,
	getExportShowUC *usecase.GetExportShowUsecase,
	createExportUC *usecase.CreateExportUsecase,
) *Handler {
	return &Handler{
		cfg:             cfg,
		flashMgr:        flashMgr,
		getExportNewUC:  getExportNewUC,
		getExportShowUC: getExportShowUC,
		createExportUC:  createExportUC,
	}
}
