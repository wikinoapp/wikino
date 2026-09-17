// Package page_backlinksはページレベルのバックリンク一覧ハンドラーを提供します
package page_backlinks

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページレベルのバックリンク一覧ハンドラー
type Handler struct {
	getPageBacklinksUC *usecase.GetPageBacklinksUsecase
}

// NewHandlerは新しいページレベルのバックリンク一覧ハンドラーを作成します
func NewHandler(
	getPageBacklinksUC *usecase.GetPageBacklinksUsecase,
) *Handler {
	return &Handler{
		getPageBacklinksUC: getPageBacklinksUC,
	}
}
