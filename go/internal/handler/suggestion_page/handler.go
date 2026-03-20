// Package suggestion_page は編集提案ページ関連のHTTPハンドラーを提供します
package suggestion_page

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// Handler は編集提案ページハンドラー
type Handler struct {
	flashMgr                    *session.FlashManager
	getSuggestionDetailUsecase  *usecase.GetSuggestionDetailUsecase
	updateSuggestionPageUsecase *usecase.UpdateSuggestionPageUsecase
	updateValidator             *validator.SuggestionPageUpdateValidator
}

// NewHandler は新しい編集提案ページハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	updateSuggestionPageUsecase *usecase.UpdateSuggestionPageUsecase,
	updateValidator *validator.SuggestionPageUpdateValidator,
) *Handler {
	return &Handler{
		flashMgr:                    flashMgr,
		getSuggestionDetailUsecase:  getSuggestionDetailUsecase,
		updateSuggestionPageUsecase: updateSuggestionPageUsecase,
		updateValidator:             updateValidator,
	}
}
