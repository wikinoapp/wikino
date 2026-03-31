// Package suggestion_apply は編集提案反映関連のHTTPハンドラーを提供します
package suggestion_apply

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案反映ハンドラー
type Handler struct {
	flashMgr               *session.FlashManager
	applySuggestionUsecase *usecase.ApplySuggestionUsecase
}

// NewHandler は新しい編集提案反映ハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	applySuggestionUsecase *usecase.ApplySuggestionUsecase,
) *Handler {
	return &Handler{
		flashMgr:               flashMgr,
		applySuggestionUsecase: applySuggestionUsecase,
	}
}
