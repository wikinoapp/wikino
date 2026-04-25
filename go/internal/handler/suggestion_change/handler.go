// Package suggestion_change は編集提案の変更差分関連のHTTPハンドラーを提供します
package suggestion_change

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案の変更差分ハンドラー
type Handler struct {
	cfg                        *config.Config
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase
	getSuggestionDiffUsecase   *usecase.GetSuggestionDiffUsecase
	sidebarHelper              *sidebar.Helper
}

// NewHandler は新しい編集提案の変更差分ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	getSuggestionDiffUsecase *usecase.GetSuggestionDiffUsecase,
	sidebarHelper *sidebar.Helper,
) *Handler {
	return &Handler{
		cfg:                        cfg,
		getSuggestionDetailUsecase: getSuggestionDetailUsecase,
		getSuggestionDiffUsecase:   getSuggestionDiffUsecase,
		sidebarHelper:              sidebarHelper,
	}
}
