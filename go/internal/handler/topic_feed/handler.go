// Package topic_feedはトピックのフィード (Atom) を配信するエンドポイントのハンドラーを提供します。
//
// フィードリーダーはログインの状態を持たずに取得するため、公開トピックのフィードだけを、
// 誰が取得しても同じ内容で返します。非公開トピックのフィードは誰が取得しても404にします。
package topic_feed

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerはトピックのフィードの配信ハンドラー
type Handler struct {
	cfg            *config.Config
	getTopicFeedUC *usecase.GetTopicFeedUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(
	cfg *config.Config,
	getTopicFeedUC *usecase.GetTopicFeedUsecase,
) *Handler {
	return &Handler{
		cfg:            cfg,
		getTopicFeedUC: getTopicFeedUC,
	}
}
