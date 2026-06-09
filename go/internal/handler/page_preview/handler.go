// Package page_preview はページ編集画面のプレビュー関連の HTTP ハンドラーを提供します
package page_preview

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はページプレビューハンドラー
type Handler struct {
	getPagePreviewUC *usecase.GetPagePreviewUsecase
}

// NewHandler は新しいページプレビューハンドラーを作成します
func NewHandler(
	getPagePreviewUC *usecase.GetPagePreviewUsecase,
) *Handler {
	return &Handler{
		getPagePreviewUC: getPagePreviewUC,
	}
}
