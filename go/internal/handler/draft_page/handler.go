// Package draft_page は下書きページ関連のHTTPハンドラーを提供します
package draft_page

import (
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は下書きページハンドラー
type Handler struct {
	spaceRepo           *repository.SpaceRepository
	spaceMemberRepo     *repository.SpaceMemberRepository
	pageRepo            *repository.PageRepository
	topicRepo           *repository.TopicRepository
	topicMemberRepo     *repository.TopicMemberRepository
	draftPageRepo       *repository.DraftPageRepository
	autoSaveDraftPageUC *usecase.AutoSaveDraftPageUsecase
}

// NewHandler は新しい下書きページハンドラーを作成します
func NewHandler(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
	autoSaveDraftPageUC *usecase.AutoSaveDraftPageUsecase,
) *Handler {
	return &Handler{
		spaceRepo:           spaceRepo,
		spaceMemberRepo:     spaceMemberRepo,
		pageRepo:            pageRepo,
		topicRepo:           topicRepo,
		topicMemberRepo:     topicMemberRepo,
		draftPageRepo:       draftPageRepo,
		autoSaveDraftPageUC: autoSaveDraftPageUC,
	}
}
