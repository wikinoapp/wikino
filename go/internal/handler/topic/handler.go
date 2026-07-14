// Package topic はトピック関連のHTTPハンドラーを提供します
package topic

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はトピックハンドラー
type Handler struct {
	cfg                   *config.Config
	flashMgr              *session.FlashManager
	getTopicDetailUsecase *usecase.GetTopicDetailUsecase
}

// NewHandler は新しいトピックハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	getTopicDetailUsecase *usecase.GetTopicDetailUsecase,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		flashMgr:              flashMgr,
		getTopicDetailUsecase: getTopicDetailUsecase,
	}
}
