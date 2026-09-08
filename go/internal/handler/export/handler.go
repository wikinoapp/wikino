// Package export provides the HTTP handlers of the space export screens.
//
// [Ja] export パッケージはスペースのエクスポート画面の HTTP ハンドラーを提供します。
package export

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler serves the screen an export is started from and the screen it is followed on.
//
// [Ja] Handler はエクスポートを開始する画面と、その経過を追う画面を提供します。
type Handler struct {
	cfg             *config.Config
	flashMgr        *session.FlashManager
	getExportNewUC  *usecase.GetExportNewUsecase
	getExportShowUC *usecase.GetExportShowUsecase
	createExportUC  *usecase.CreateExportUsecase
}

// NewHandler creates an export Handler.
//
// [Ja] NewHandler はエクスポートのハンドラーを生成します。
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
