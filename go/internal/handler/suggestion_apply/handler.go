// Package suggestion_apply は編集提案反映関連のHTTPハンドラーを提供します
package suggestion_apply

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は編集提案反映ハンドラー
type Handler struct {
	cfg                        *config.Config
	flashMgr                   *session.FlashManager
	applySuggestionUsecase     *usecase.ApplySuggestionUsecase
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase
}

// NewHandler は新しい編集提案反映ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	applySuggestionUsecase *usecase.ApplySuggestionUsecase,
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase,
) *Handler {
	return &Handler{
		cfg:                        cfg,
		flashMgr:                   flashMgr,
		applySuggestionUsecase:     applySuggestionUsecase,
		getSuggestionDetailUsecase: getSuggestionDetailUsecase,
	}
}
