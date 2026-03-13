// Package page_backlinks はページレベルのバックリンク一覧SSEハンドラーを提供します
package page_backlinks

import (
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はページレベルのバックリンク一覧ハンドラー
type Handler struct {
	getPageBacklinksUC *usecase.GetPageBacklinksUsecase
}

// NewHandler は新しいページレベルのバックリンク一覧ハンドラーを作成します
func NewHandler(
	getPageBacklinksUC *usecase.GetPageBacklinksUsecase,
) *Handler {
	return &Handler{
		getPageBacklinksUC: getPageBacklinksUC,
	}
}
