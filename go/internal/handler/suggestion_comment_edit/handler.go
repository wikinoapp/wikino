// Package suggestion_comment_edit は編集提案コメント編集関連のHTTPハンドラーを提供します
package suggestion_comment_edit

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler は編集提案コメント編集ハンドラー
type Handler struct {
	cfg                            *config.Config
	flashMgr                       *session.FlashManager
	getSuggestionDetailUsecase     *usecase.GetSuggestionDetailUsecase
	getSuggestionCommentUsecase    *usecase.GetSuggestionCommentUsecase
	updateSuggestionCommentUsecase *usecase.UpdateSuggestionCommentUsecase
	sidebarHelper                  *sidebar.Helper
	updateValidator                *validator.SuggestionCommentUpdateValidator
}

// NewHandler は新しい編集提案コメント編集ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	getSuggestionCommentUsecase *usecase.GetSuggestionCommentUsecase,
	updateSuggestionCommentUsecase *usecase.UpdateSuggestionCommentUsecase,
	sidebarHelper *sidebar.Helper,
	updateValidator *validator.SuggestionCommentUpdateValidator,
) *Handler {
	return &Handler{
		cfg:                            cfg,
		flashMgr:                       flashMgr,
		getSuggestionDetailUsecase:     getSuggestionDetailUsecase,
		getSuggestionCommentUsecase:    getSuggestionCommentUsecase,
		updateSuggestionCommentUsecase: updateSuggestionCommentUsecase,
		sidebarHelper:                  sidebarHelper,
		updateValidator:                updateValidator,
	}
}
