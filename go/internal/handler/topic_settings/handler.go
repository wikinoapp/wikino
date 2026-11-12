// Package topic_settingsはトピックの設定のトップのHTTPハンドラーを提供します。
package topic_settings

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはトピックの設定のトップを提供します。
type Handler struct {
	cfg                *config.Config
	getTopicSettingsUC *usecase.GetTopicSettingsUsecase
}

// NewHandlerはトピックの設定のトップのハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	getTopicSettingsUC *usecase.GetTopicSettingsUsecase,
) *Handler {
	return &Handler{
		cfg:                cfg,
		getTopicSettingsUC: getTopicSettingsUC,
	}
}
