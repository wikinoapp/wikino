// Package suggestion_comment は編集提案コメント作成関連のHTTPハンドラーを提供します
package suggestion_comment

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案コメント作成ハンドラー
type Handler struct {
	flashMgr                       *session.FlashManager
	createSuggestionCommentUsecase *usecase.CreateSuggestionCommentUsecase
}

// NewHandler は新しい編集提案コメント作成ハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	createSuggestionCommentUsecase *usecase.CreateSuggestionCommentUsecase,
) *Handler {
	return &Handler{
		flashMgr:                       flashMgr,
		createSuggestionCommentUsecase: createSuggestionCommentUsecase,
	}
}
