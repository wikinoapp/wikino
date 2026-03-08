// Package page はページ関連のHTTPハンドラーを提供します
package page

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はページハンドラー
type Handler struct {
	cfg             *config.Config
	flashMgr        *session.FlashManager
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	draftPageRepo   *repository.DraftPageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	publishPageUC   *usecase.PublishPageUsecase
	sidebarHelper   *sidebar.Helper
}

// NewHandler は新しいページハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	draftPageRepo *repository.DraftPageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	publishPageUC *usecase.PublishPageUsecase,
	sidebarHelper *sidebar.Helper,
) *Handler {
	return &Handler{
		cfg:             cfg,
		flashMgr:        flashMgr,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		draftPageRepo:   draftPageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		publishPageUC:   publishPageUC,
		sidebarHelper:   sidebarHelper,
	}
}
