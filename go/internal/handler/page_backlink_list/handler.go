// Package page_backlink_listはページのバックリンク一覧ハンドラーを提供します
package page_backlink_list

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはバックリンク一覧ハンドラー
type Handler struct {
	getBacklinkListUC *usecase.GetBacklinkListUsecase
}

// NewHandlerは新しいバックリンク一覧ハンドラーを作成します
func NewHandler(
	getBacklinkListUC *usecase.GetBacklinkListUsecase,
) *Handler {
	return &Handler{
		getBacklinkListUC: getBacklinkListUC,
	}
}
