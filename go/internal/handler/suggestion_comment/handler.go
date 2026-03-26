// Package suggestion_comment は編集提案コメント作成関連のHTTPハンドラーを提供します
package suggestion_comment

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler は編集提案コメント作成ハンドラー
type Handler struct {
	flashMgr                       *session.FlashManager
	getSuggestionDetailUsecase     *usecase.GetSuggestionDetailUsecase
	createSuggestionCommentUsecase *usecase.CreateSuggestionCommentUsecase
	createValidator                *validator.SuggestionCommentCreateValidator
}

// NewHandler は新しい編集提案コメント作成ハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	createSuggestionCommentUsecase *usecase.CreateSuggestionCommentUsecase,
	createValidator *validator.SuggestionCommentCreateValidator,
) *Handler {
	return &Handler{
		flashMgr:                       flashMgr,
		getSuggestionDetailUsecase:     getSuggestionDetailUsecase,
		createSuggestionCommentUsecase: createSuggestionCommentUsecase,
		createValidator:                createValidator,
	}
}
