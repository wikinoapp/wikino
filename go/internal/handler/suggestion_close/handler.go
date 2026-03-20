// Package suggestion_close は編集提案クローズ関連のHTTPハンドラーを提供します
package suggestion_close

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案クローズハンドラー
type Handler struct {
	flashMgr                   *session.FlashManager
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase
	closeSuggestionUsecase     *usecase.CloseSuggestionUsecase
}

// NewHandler は新しい編集提案クローズハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	closeSuggestionUsecase *usecase.CloseSuggestionUsecase,
) *Handler {
	return &Handler{
		flashMgr:                   flashMgr,
		getSuggestionDetailUsecase: getSuggestionDetailUsecase,
		closeSuggestionUsecase:     closeSuggestionUsecase,
	}
}
