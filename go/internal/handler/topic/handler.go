// Package topic はトピック関連のHTTPハンドラーを提供します
package topic

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
)

// Handler はトピックハンドラー
type Handler struct {
	cfg             *config.Config
	flashMgr        *session.FlashManager
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	pageRepo        *repository.PageRepository
	sidebarHelper   *sidebar.Helper
}

// NewHandler は新しいトピックハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	pageRepo *repository.PageRepository,
	sidebarHelper *sidebar.Helper,
) *Handler {
	return &Handler{
		cfg:             cfg,
		flashMgr:        flashMgr,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		pageRepo:        pageRepo,
		sidebarHelper:   sidebarHelper,
	}
}
