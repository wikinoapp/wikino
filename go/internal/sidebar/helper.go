// Package sidebar はサイドバーコンテンツの取得を提供します
package sidebar

import (
	"context"
	"log/slog"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Helper はサイドバーに表示するデータの取得を担当します
type Helper struct {
	topicRepo     *repository.TopicRepository
	draftPageRepo *repository.DraftPageRepository
}

// NewHelper は新しいサイドバーヘルパーを作成します
func NewHelper(
	topicRepo *repository.TopicRepository,
	draftPageRepo *repository.DraftPageRepository,
) *Helper {
	return &Helper{
		topicRepo:     topicRepo,
		draftPageRepo: draftPageRepo,
	}
}

// SidebarContent はサイドバーに表示するデータです
type SidebarContent struct {
	JoinedTopics      []viewmodel.TopicForSidebar
	DraftPages        []viewmodel.DraftPageForSidebar
	HasMoreDraftPages bool
}

// Content はサイドバーに表示する参加中トピックと下書きページを取得します。
// エラー時は空スライスを返し、ページ表示は継続します。
func (h *Helper) Content(ctx context.Context, userID model.UserID) SidebarContent {
	topics, err := h.topicRepo.ListJoinedByUser(ctx, userID, viewmodel.SidebarJoinedTopicsLimit)
	if err != nil {
		slog.ErrorContext(ctx, "サイドバー用の参加中トピック取得に失敗", "error", err)
		topics = nil
	}

	// limit+1件取得し、超過分があればhasMoreとする
	drafts, err := h.draftPageRepo.ListByUser(ctx, userID, viewmodel.SidebarDraftPagesLimit+1)
	if err != nil {
		slog.ErrorContext(ctx, "サイドバー用の下書きページ取得に失敗", "error", err)
		drafts = nil
	}

	hasMore := len(drafts) > int(viewmodel.SidebarDraftPagesLimit)
	if hasMore {
		drafts = drafts[:viewmodel.SidebarDraftPagesLimit]
	}

	return SidebarContent{
		JoinedTopics:      viewmodel.NewTopicsForSidebar(topics),
		DraftPages:        viewmodel.NewDraftPagesForSidebar(drafts),
		HasMoreDraftPages: hasMore,
	}
}
