// Package page_link_list はページのリンク一覧SSEハンドラーを提供します
package page_link_list

import (
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// Handler はリンク一覧ハンドラー
type Handler struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	draftPageRepo   *repository.DraftPageRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewHandler は新しいリンク一覧ハンドラーを作成します
func NewHandler(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	draftPageRepo *repository.DraftPageRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *Handler {
	return &Handler{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		draftPageRepo:   draftPageRepo,
		topicMemberRepo: topicMemberRepo,
	}
}
