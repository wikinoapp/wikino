// Package page_link_list はページのリンク一覧SSEハンドラーを提供します
package page_link_list

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はリンク一覧ハンドラー
type Handler struct {
	getLinkListUC *usecase.GetLinkListUsecase
}

// NewHandler は新しいリンク一覧ハンドラーを作成します
func NewHandler(
	getLinkListUC *usecase.GetLinkListUsecase,
) *Handler {
	return &Handler{
		getLinkListUC: getLinkListUC,
	}
}
