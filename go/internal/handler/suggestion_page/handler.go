// Package suggestion_page は編集提案ページ関連のHTTPハンドラーを提供します
package suggestion_page

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案ページハンドラー
type Handler struct {
	cfg                         *config.Config
	flashMgr                    *session.FlashManager
	getSuggestionPageNewUsecase *usecase.GetSuggestionPageNewUsecase
	addSuggestionPageUsecase    *usecase.AddSuggestionPageUsecase
	updateSuggestionPageUsecase *usecase.UpdateSuggestionPageUsecase
	removeSuggestionPageUsecase *usecase.RemoveSuggestionPageUsecase
	sidebarHelper               *sidebar.Helper
}

// NewHandler は新しい編集提案ページハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getSuggestionPageNewUsecase *usecase.GetSuggestionPageNewUsecase,
	addSuggestionPageUsecase *usecase.AddSuggestionPageUsecase,
	updateSuggestionPageUsecase *usecase.UpdateSuggestionPageUsecase,
	removeSuggestionPageUsecase *usecase.RemoveSuggestionPageUsecase,
	sidebarHelper *sidebar.Helper,
) *Handler {
	return &Handler{
		cfg:                         cfg,
		flashMgr:                    flashMgr,
		getSuggestionPageNewUsecase: getSuggestionPageNewUsecase,
		addSuggestionPageUsecase:    addSuggestionPageUsecase,
		updateSuggestionPageUsecase: updateSuggestionPageUsecase,
		removeSuggestionPageUsecase: removeSuggestionPageUsecase,
		sidebarHelper:               sidebarHelper,
	}
}
