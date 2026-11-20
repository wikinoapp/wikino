// Package bulk_restored_pageは、ゴミ箱で選んだページをまとめて復元するHTTPハンドラーを提供します。
package bulk_restored_page

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはゴミ箱からの一括復元を提供します。
type Handler struct {
	flashMgr       *session.FlashManager
	restorePagesUC *usecase.RestorePagesUsecase
}

// NewHandlerは一括復元のハンドラーを生成します
func NewHandler(
	flashMgr *session.FlashManager,
	restorePagesUC *usecase.RestorePagesUsecase,
) *Handler {
	return &Handler{
		flashMgr:       flashMgr,
		restorePagesUC: restorePagesUC,
	}
}
