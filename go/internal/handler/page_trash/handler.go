// Package page_trashはページをゴミ箱へ入れる操作のHTTPハンドラーを提供します。
package page_trash

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページをゴミ箱へ入れるリクエストを処理する。
type Handler struct {
	flashMgr    *session.FlashManager
	trashPageUC *usecase.TrashPageUsecase
}

// NewHandlerはHandlerを生成する。
func NewHandler(
	flashMgr *session.FlashManager,
	trashPageUC *usecase.TrashPageUsecase,
) *Handler {
	return &Handler{
		flashMgr:    flashMgr,
		trashPageUC: trashPageUC,
	}
}
