// Package suggestion_changeは編集提案の変更差分関連のHTTPハンドラーを提供します
package suggestion_change

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは編集提案の変更差分ハンドラー
type Handler struct {
	cfg                        *config.Config
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase
	getSuggestionDiffUsecase   *usecase.GetSuggestionDiffUsecase
}

// NewHandlerは新しい編集提案の変更差分ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
	getSuggestionDiffUsecase *usecase.GetSuggestionDiffUsecase,
) *Handler {
	return &Handler{
		cfg:                        cfg,
		getSuggestionDetailUsecase: getSuggestionDetailUsecase,
		getSuggestionDiffUsecase:   getSuggestionDiffUsecase,
	}
}
