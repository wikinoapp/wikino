// Package suggestion_closureは編集提案をクローズする操作のHTTPハンドラーを提供します。
package suggestion_closure

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは編集提案クローズハンドラー
type Handler struct {
	flashMgr               *session.FlashManager
	closeSuggestionUsecase *usecase.CloseSuggestionUsecase
}

// NewHandlerは新しい編集提案クローズハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	closeSuggestionUsecase *usecase.CloseSuggestionUsecase,
) *Handler {
	return &Handler{
		flashMgr:               flashMgr,
		closeSuggestionUsecase: closeSuggestionUsecase,
	}
}
