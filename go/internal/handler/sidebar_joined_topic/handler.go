// Package sidebar_joined_topic はサイドバーの参加中トピック一覧SSEハンドラーを提供します
package sidebar_joined_topic

import (
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// Handler はサイドバーの参加中トピック一覧ハンドラー
type Handler struct {
	topicRepo *repository.TopicRepository
}

// NewHandler は新しいサイドバー参加中トピック一覧ハンドラーを作成します
func NewHandler(topicRepo *repository.TopicRepository) *Handler {
	return &Handler{
		topicRepo: topicRepo,
	}
}
