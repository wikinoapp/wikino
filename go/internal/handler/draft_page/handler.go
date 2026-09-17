// Package draft_pageは下書きページ関連のHTTPハンドラーを提供します
package draft_page

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは下書きページハンドラー
type Handler struct {
	flashMgr            *session.FlashManager
	getPageDetailUC     *usecase.GetPageDetailUsecase
	autoSaveDraftPageUC *usecase.AutoSaveDraftPageUsecase
	deleteDraftPageUC   *usecase.DeleteDraftPageUsecase
	getEditLinkDataUC   *usecase.GetEditLinkDataUsecase
}

// NewHandlerは新しい下書きページハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	getPageDetailUC *usecase.GetPageDetailUsecase,
	autoSaveDraftPageUC *usecase.AutoSaveDraftPageUsecase,
	deleteDraftPageUC *usecase.DeleteDraftPageUsecase,
	getEditLinkDataUC *usecase.GetEditLinkDataUsecase,
) *Handler {
	return &Handler{
		flashMgr:            flashMgr,
		getPageDetailUC:     getPageDetailUC,
		autoSaveDraftPageUC: autoSaveDraftPageUC,
		deleteDraftPageUC:   deleteDraftPageUC,
		getEditLinkDataUC:   getEditLinkDataUC,
	}
}
