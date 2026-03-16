// Package suggestion は編集提案関連のHTTPハンドラーを提供します
package suggestion

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案ハンドラー
type Handler struct {
	cfg                      *config.Config
	getSuggestionListUsecase *usecase.GetSuggestionListUsecase
	sidebarHelper            *sidebar.Helper
}

// NewHandler は新しい編集提案ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	getSuggestionListUsecase *usecase.GetSuggestionListUsecase,
	sidebarHelper *sidebar.Helper,
) *Handler {
	return &Handler{
		cfg:                      cfg,
		getSuggestionListUsecase: getSuggestionListUsecase,
		sidebarHelper:            sidebarHelper,
	}
}
