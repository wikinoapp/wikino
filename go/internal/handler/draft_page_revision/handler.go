// Package draft_page_revision は下書きリビジョン関連のHTTPハンドラーを提供します
package draft_page_revision

import (
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は下書きリビジョンハンドラー
type Handler struct {
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	pageRepo              *repository.PageRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	flashMgr              *session.FlashManager
	manualSaveDraftPageUC *usecase.ManualSaveDraftPageUsecase
}

// NewHandler は新しい下書きリビジョンハンドラーを作成します
func NewHandler(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	flashMgr *session.FlashManager,
	manualSaveDraftPageUC *usecase.ManualSaveDraftPageUsecase,
) *Handler {
	return &Handler{
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		pageRepo:              pageRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		flashMgr:              flashMgr,
		manualSaveDraftPageUC: manualSaveDraftPageUC,
	}
}
