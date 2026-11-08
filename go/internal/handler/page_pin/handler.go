// Package page_pinはページにピンを打つ・外す操作のHTTPハンドラーを提供します。
package page_pin

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはページにピンを打つ・外すリクエストを処理する。
type Handler struct {
	flashMgr    *session.FlashManager
	pinPageUC   *usecase.PinPageUsecase
	unpinPageUC *usecase.UnpinPageUsecase
}

// NewHandlerはHandlerを生成する。
func NewHandler(
	flashMgr *session.FlashManager,
	pinPageUC *usecase.PinPageUsecase,
	unpinPageUC *usecase.UnpinPageUsecase,
) *Handler {
	return &Handler{
		flashMgr:    flashMgr,
		pinPageUC:   pinPageUC,
		unpinPageUC: unpinPageUC,
	}
}
