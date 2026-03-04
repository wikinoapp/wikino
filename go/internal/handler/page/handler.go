// Package page はページ関連のHTTPハンドラーを提供します
package page

import (
	"context"
	"log/slog"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// サイドバーに表示する参加中トピックの最大件数
const sidebarJoinedTopicsLimit = 10

// サイドバーに表示する下書きページの最大件数
const sidebarDraftPagesLimit = 5

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
	}
}

// sidebarContent はサイドバーに表示する参加中トピックと下書きページを取得します。
// エラー時は空スライスを返し、ページ表示は継続します。
func (h *Handler) sidebarContent(ctx context.Context, userID model.UserID) ([]viewmodel.TopicForSidebar, []viewmodel.DraftPageForSidebar) {
	topics, err := h.topicRepo.ListJoinedByUser(ctx, userID, sidebarJoinedTopicsLimit)
	if err != nil {
		slog.ErrorContext(ctx, "サイドバー用の参加中トピック取得に失敗", "error", err)
		topics = nil
	}

	drafts, err := h.draftPageRepo.ListByUser(ctx, userID, sidebarDraftPagesLimit)
	if err != nil {
		slog.ErrorContext(ctx, "サイドバー用の下書きページ取得に失敗", "error", err)
		drafts = nil
	}

	return viewmodel.NewTopicsForSidebar(topics), viewmodel.NewDraftPagesForSidebar(drafts)
}
