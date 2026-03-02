package sidebar_joined_topic

import (
	"log/slog"
	"net/http"

	datastar "github.com/starfederation/datastar-go/datastar"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// サイドバーに表示する参加中トピックの最大件数
const joinedTopicsLimit = 10

// Index は参加中のトピック一覧をSSEフラグメントとして返します (GET /sidebar/joined_topics)
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		sse := datastar.NewSSE(w, r)
		if err := sse.PatchElementTempl(components.SidebarJoinedTopics(nil), datastar.WithSelectorID("sidebar-joined-topics"), datastar.WithModeInner()); err != nil {
			slog.ErrorContext(ctx, "サイドバートピック一覧の空SSE送信に失敗", "error", err)
		}
		return
	}

	topics, err := h.topicRepo.ListJoinedByUser(ctx, user.ID, joinedTopicsLimit)
	if err != nil {
		slog.ErrorContext(ctx, "参加中トピック一覧の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	topicsVM := viewmodel.NewTopicsForSidebar(topics)

	sse := datastar.NewSSE(w, r)
	if err := sse.PatchElementTempl(components.SidebarJoinedTopics(topicsVM), datastar.WithSelectorID("sidebar-joined-topics"), datastar.WithModeInner()); err != nil {
		slog.ErrorContext(ctx, "サイドバートピック一覧のSSE送信に失敗", "error", err)
		return
	}
}
