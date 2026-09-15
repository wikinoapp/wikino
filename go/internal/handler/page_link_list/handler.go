// Package page_link_listはページのリンク一覧ハンドラーを提供します
package page_link_list

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはリンク一覧ハンドラー
type Handler struct {
	getLinkListUC *usecase.GetLinkListUsecase
}

// NewHandlerは新しいリンク一覧ハンドラーを作成します
func NewHandler(
	getLinkListUC *usecase.GetLinkListUsecase,
) *Handler {
	return &Handler{
		getLinkListUC: getLinkListUC,
	}
}
