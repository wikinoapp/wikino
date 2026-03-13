// Package page_backlink_list はページのバックリンク一覧SSEハンドラーを提供します
package page_backlink_list

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はバックリンク一覧ハンドラー
type Handler struct {
	getBacklinkListUC *usecase.GetBacklinkListUsecase
}

// NewHandler は新しいバックリンク一覧ハンドラーを作成します
func NewHandler(
	getBacklinkListUC *usecase.GetBacklinkListUsecase,
) *Handler {
	return &Handler{
		getBacklinkListUC: getBacklinkListUC,
	}
}
