// Package suggestion_closure provides the HTTP handlers for closing a suggestion.
//
// [Ja] Package suggestion_closure は編集提案をクローズする操作の HTTP ハンドラーを提供します。
package suggestion_closure

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案クローズハンドラー
type Handler struct {
	flashMgr               *session.FlashManager
	closeSuggestionUsecase *usecase.CloseSuggestionUsecase
}

// NewHandler は新しい編集提案クローズハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	closeSuggestionUsecase *usecase.CloseSuggestionUsecase,
) *Handler {
	return &Handler{
		flashMgr:               flashMgr,
		closeSuggestionUsecase: closeSuggestionUsecase,
	}
}
