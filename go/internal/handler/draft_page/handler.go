// Package draft_page は下書きページ関連のHTTPハンドラーを提供します
package draft_page

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は下書きページハンドラー
type Handler struct {
	flashMgr            *session.FlashManager
	getPageDetailUC     *usecase.GetPageDetailUsecase
	autoSaveDraftPageUC *usecase.AutoSaveDraftPageUsecase
	deleteDraftPageUC   *usecase.DeleteDraftPageUsecase
	getEditLinkDataUC   *usecase.GetEditLinkDataUsecase
}

// NewHandler は新しい下書きページハンドラーを作成します
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
