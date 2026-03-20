// Package suggestion_page_revision は編集提案ページリビジョン関連のHTTPハンドラーを提供します
package suggestion_page_revision

import (
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案ページリビジョンハンドラー
type Handler struct {
	flashMgr                    *session.FlashManager
	getSuggestionDetailUsecase  *usecase.GetSuggestionDetailUsecase
	updateSuggestionPageUsecase *usecase.UpdateSuggestionPageUsecase
}

// NewHandler は新しい編集提案ページリビジョンハンドラーを作成します
func NewHandler(
	flashMgr *session.FlashManager,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	updateSuggestionPageUsecase *usecase.UpdateSuggestionPageUsecase,
) *Handler {
	return &Handler{
		flashMgr:                    flashMgr,
		getSuggestionDetailUsecase:  getSuggestionDetailUsecase,
		updateSuggestionPageUsecase: updateSuggestionPageUsecase,
	}
}
