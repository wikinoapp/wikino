// Package draft_page は下書きページ関連のHTTPハンドラーを提供します
package draft_page

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は下書きページハンドラー
type Handler struct {
	getPageDetailUC     *usecase.GetPageDetailUsecase
	autoSaveDraftPageUC *usecase.AutoSaveDraftPageUsecase
	getEditLinkDataUC   *usecase.GetEditLinkDataUsecase
}

// NewHandler は新しい下書きページハンドラーを作成します
func NewHandler(
	getPageDetailUC *usecase.GetPageDetailUsecase,
	autoSaveDraftPageUC *usecase.AutoSaveDraftPageUsecase,
	getEditLinkDataUC *usecase.GetEditLinkDataUsecase,
) *Handler {
	return &Handler{
		getPageDetailUC:     getPageDetailUC,
		autoSaveDraftPageUC: autoSaveDraftPageUC,
		getEditLinkDataUC:   getEditLinkDataUC,
	}
}
