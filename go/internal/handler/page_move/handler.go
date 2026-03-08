// Package page_move はページ移動関連のHTTPハンドラーを提供します
package page_move

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler はページ移動ハンドラー
type Handler struct {
	cfg             *config.Config
	flashMgr        *session.FlashManager
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	movePageUC      *usecase.MovePageUsecase
	sidebarHelper   *sidebar.Helper
}

// NewHandler は新しいページ移動ハンドラーを作成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	movePageUC *usecase.MovePageUsecase,
	sidebarHelper *sidebar.Helper,
) *Handler {
	return &Handler{
		cfg:             cfg,
		flashMgr:        flashMgr,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		movePageUC:      movePageUC,
		sidebarHelper:   sidebarHelper,
	}
}
