// Package space_settingsはスペース設定のトップのHTTPハンドラーを提供します。
package space_settings

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはスペース設定のトップを提供します。
type Handler struct {
	cfg                *config.Config
	getSpaceSettingsUC *usecase.GetSpaceSettingsUsecase
}

// NewHandlerはスペース設定のトップのハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	getSpaceSettingsUC *usecase.GetSpaceSettingsUsecase,
) *Handler {
	return &Handler{
		cfg:                cfg,
		getSpaceSettingsUC: getSpaceSettingsUC,
	}
}
