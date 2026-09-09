// Package suggestion_comment_edit は編集提案コメント編集関連のHTTPハンドラーを提供します
package suggestion_comment_edit

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案コメント編集ハンドラー
type Handler struct {
	cfg                            *config.Config
	flashMgr                       *session.FlashManager
	getSuggestionEditUsecase       *usecase.GetSuggestionEditUsecase
	getSuggestionCommentUsecase    *usecase.GetSuggestionCommentUsecase
	updateSuggestionCommentUsecase *usecase.UpdateSuggestionCommentUsecase
}

// NewHandler は新しい編集提案コメント編集ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSuggestionEditUsecase *usecase.GetSuggestionEditUsecase,
	getSuggestionCommentUsecase *usecase.GetSuggestionCommentUsecase,
	updateSuggestionCommentUsecase *usecase.UpdateSuggestionCommentUsecase,
) *Handler {
	return &Handler{
		cfg:                            cfg,
		flashMgr:                       flashMgr,
		getSuggestionEditUsecase:       getSuggestionEditUsecase,
		getSuggestionCommentUsecase:    getSuggestionCommentUsecase,
		updateSuggestionCommentUsecase: updateSuggestionCommentUsecase,
	}
}
