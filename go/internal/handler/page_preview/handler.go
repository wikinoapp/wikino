// Package page_previewはページ編集画面のプレビュー関連のHTTPハンドラーを提供します
package page_preview

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページプレビューハンドラー
type Handler struct {
	getPagePreviewUC *usecase.GetPagePreviewUsecase
}

// NewHandlerは新しいページプレビューハンドラーを作成します
func NewHandler(
	getPagePreviewUC *usecase.GetPagePreviewUsecase,
) *Handler {
	return &Handler{
		getPagePreviewUC: getPagePreviewUC,
	}
}
