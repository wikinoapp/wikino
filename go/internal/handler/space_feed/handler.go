// Package space_feedはスペースのフィード (Atom) を配信するエンドポイントのハンドラーを提供します。
//
// フィードリーダーはログインの状態を持たずに取得するため、誰が取得しても
// 公開トピックのページだけを載せた同じ内容を返します。
package space_feed

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはスペースのフィードの配信ハンドラー
type Handler struct {
	cfg            *config.Config
	getSpaceFeedUC *usecase.GetSpaceFeedUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(
	cfg *config.Config,
	getSpaceFeedUC *usecase.GetSpaceFeedUsecase,
) *Handler {
	return &Handler{
		cfg:            cfg,
		getSpaceFeedUC: getSpaceFeedUC,
	}
}
