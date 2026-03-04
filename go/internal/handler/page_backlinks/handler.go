// Package page_backlinks はページレベルのバックリンク一覧SSEハンドラーを提供します
package page_backlinks

import (
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// Handler はページレベルのバックリンク一覧ハンドラー
type Handler struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewHandler は新しいページレベルのバックリンク一覧ハンドラーを作成します
func NewHandler(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *Handler {
	return &Handler{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}
