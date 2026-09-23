// Package suggestion_applicationは編集提案を反映する操作のHTTPハンドラーを提供します。
package suggestion_application

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは編集提案反映ハンドラー
type Handler struct {
	cfg                        *config.Config
	flashMgr                   *session.FlashManager
	applySuggestionUsecase     *usecase.ApplySuggestionUsecase
	getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase
}

// NewHandlerは新しい編集提案反映ハンドラーを作成します
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
