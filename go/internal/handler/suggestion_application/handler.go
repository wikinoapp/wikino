// Package suggestion_application provides the HTTP handlers for applying a suggestion.
//
// [Ja] Package suggestion_application は編集提案を反映する操作の HTTP ハンドラーを提供します。
package suggestion_application

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
