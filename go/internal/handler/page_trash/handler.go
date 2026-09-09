// Package page_trash provides the HTTP handlers for moving a page into the trash.
//
// [Ja] Package page_trash はページをゴミ箱へ入れる操作の HTTP ハンドラーを提供します。
package page_trash

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler handles requests that move pages into the trash.
//
// [Ja] Handler はページをゴミ箱へ入れるリクエストを処理する。
type Handler struct {
	flashMgr    *session.FlashManager
	trashPageUC *usecase.TrashPageUsecase
}

// NewHandler creates a Handler.
//
// [Ja] NewHandler は Handler を生成する。
func NewHandler(
	flashMgr *session.FlashManager,
	trashPageUC *usecase.TrashPageUsecase,
) *Handler {
	return &Handler{
		flashMgr:    flashMgr,
		trashPageUC: trashPageUC,
	}
}
