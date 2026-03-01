// Package page_backlink_list はページのバックリンク一覧SSEハンドラーを提供します
package page_backlink_list

import (
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// Handler はバックリンク一覧ハンドラー
type Handler struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewHandler は新しいバックリンク一覧ハンドラーを作成します
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
