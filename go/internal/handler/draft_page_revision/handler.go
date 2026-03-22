// Package draft_page_revision は下書きリビジョン関連のHTTPハンドラーを提供します
package draft_page_revision

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は下書きリビジョンハンドラー
type Handler struct {
	getPageDetailUC       *usecase.GetPageDetailUsecase
	flashMgr              *session.FlashManager
	manualSaveDraftPageUC *usecase.ManualSaveDraftPageUsecase
}

// NewHandler は新しい下書きリビジョンハンドラーを作成します
func NewHandler(
	getPageDetailUC *usecase.GetPageDetailUsecase,
	flashMgr *session.FlashManager,
	manualSaveDraftPageUC *usecase.ManualSaveDraftPageUsecase,
) *Handler {
	return &Handler{
		getPageDetailUC:       getPageDetailUC,
		flashMgr:              flashMgr,
		manualSaveDraftPageUC: manualSaveDraftPageUC,
	}
}
