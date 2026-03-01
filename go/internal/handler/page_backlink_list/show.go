package page_backlink_list

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar-go/datastar"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show はバックリンク一覧をSSEフラグメントとして返します (GET /go/s/{space_identifier}/pages/{page_number}/links/{linked_page_number}/backlink_list)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	pageNumberStr := chi.URLParam(r, "page_number")
	linkedPageNumberStr := chi.URLParam(r, "linked_page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	linkedPageNumber, err := strconv.ParseInt(linkedPageNumberStr, 10, 32)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// スペースを取得
	space, err := h.spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		slog.ErrorContext(ctx, "スペースの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if space == nil {
		http.NotFound(w, r)
		return
	}

	// スペースメンバーを取得
	spaceMember, err := h.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, user.ID)
	if err != nil {
		slog.ErrorContext(ctx, "スペースメンバーの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if spaceMember == nil {
		http.NotFound(w, r)
		return
	}

	// ページを取得（編集中のページ）
	pg, err := h.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(pageNumber))
	if err != nil {
		slog.ErrorContext(ctx, "ページの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if pg == nil {
		http.NotFound(w, r)
		return
	}

	// トピックメンバーを取得してTopicPolicyを生成
	topicMember, err := h.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, pg.TopicID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックメンバーの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	topicPolicy := policy.NewTopicPolicy(spaceMember, topicMember)
	if !topicPolicy.CanUpdatePage(pg) {
		http.NotFound(w, r)
		return
	}

	// リンク先ページ（バックリンクの対象）を取得
	linkedPage, err := h.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(linkedPageNumber))
	if err != nil {
		slog.ErrorContext(ctx, "リンク先ページの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if linkedPage == nil {
		http.NotFound(w, r)
		return
	}

	// ページネーションパラメータを取得
	currentPage := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			currentPage = int32(p)
		}
	}

	// バックリンクを取得
	paginatedBacklinks, err := h.pageRepo.FindBacklinkedPagesPaginated(ctx, linkedPage.ID, space.ID, currentPage, viewmodel.BacklinkLimit)
	if err != nil {
		slog.ErrorContext(ctx, "バックリンクの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// バックリンクページからTopicIDを収集してトピックを一括取得
	topicIDSet := make(map[model.TopicID]struct{})
	for _, p := range paginatedBacklinks.Pages {
		topicIDSet[p.TopicID] = struct{}{}
	}

	topicIDs := make([]model.TopicID, 0, len(topicIDSet))
	for id := range topicIDSet {
		topicIDs = append(topicIDs, id)
	}

	topics, err := h.topicRepo.FindByIDsAndSpace(ctx, topicIDs, space.ID)
	if err != nil {
		slog.ErrorContext(ctx, "トピックの一括取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	topicMap := make(map[model.TopicID]*model.Topic, len(topics))
	for _, t := range topics {
		topicMap[t.ID] = t
	}

	backlinkListVM := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:            paginatedBacklinks.Pages,
		TopicMap:         topicMap,
		Pagination:       viewmodel.NewPagination(int(currentPage), paginatedBacklinks.TotalCount, int(viewmodel.BacklinkLimit)),
		SpaceIdentifier:  spaceIdentifier,
		PageNumber:       int32(pageNumber),
		LinkedPageNumber: int32(linkedPageNumber),
	})

	// SSEフラグメントとしてバックリンク一覧を送信
	selectorID := fmt.Sprintf("page-backlink-list-%d", linkedPageNumber)
	sse := datastar.NewSSE(w, r)
	if err := sse.PatchElementTempl(components.BacklinkList(backlinkListVM), datastar.WithSelectorID(selectorID), datastar.WithModeInner()); err != nil {
		slog.ErrorContext(ctx, "バックリンク一覧のSSE送信に失敗", "error", err)
		return
	}
}
