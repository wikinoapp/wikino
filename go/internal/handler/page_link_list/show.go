package page_link_list

import (
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

// Show はリンク一覧の追加ページをSSEフラグメントとして返します (GET /s/{space_identifier}/pages/{page_number}/link_list)
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

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
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

	// ページを取得
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

	// DraftPageを取得してリンク先ページIDを決定
	draftPage, err := h.draftPageRepo.FindByPageAndMember(ctx, pg.ID, spaceMember.ID, space.ID)
	if err != nil {
		slog.ErrorContext(ctx, "下書きの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var linkedPageIDs []model.PageID
	if draftPage != nil {
		linkedPageIDs = draftPage.LinkedPageIDs
	} else {
		linkedPageIDs = pg.LinkedPageIDs
	}

	// ページネーションパラメータを取得
	currentPage := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			currentPage = int32(p)
		}
	}

	if len(linkedPageIDs) == 0 {
		return
	}

	// リンク先ページを取得
	paginatedLinks, err := h.pageRepo.FindLinkedPagesPaginated(ctx, linkedPageIDs, space.ID, currentPage, viewmodel.LinkLimit)
	if err != nil {
		slog.ErrorContext(ctx, "リンク先ページの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 編集中のページ自身とリンク先ページをバックリンクから除外する
	excludePageIDs := make([]model.PageID, 0, 1+len(paginatedLinks.Pages))
	excludePageIDs = append(excludePageIDs, pg.ID)
	for _, p := range paginatedLinks.Pages {
		excludePageIDs = append(excludePageIDs, p.ID)
	}

	backlinkPaginatedMap, err := h.pageRepo.FindBacklinksForPages(ctx, paginatedLinks.Pages, space.ID, viewmodel.BacklinkLimit, excludePageIDs)
	if err != nil {
		slog.ErrorContext(ctx, "バックリンクの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// すべてのページからTopicIDを収集してトピックを一括取得
	topicIDSet := make(map[model.TopicID]struct{})
	for _, p := range paginatedLinks.Pages {
		topicIDSet[p.TopicID] = struct{}{}
	}
	for _, paginated := range backlinkPaginatedMap {
		for _, p := range paginated.Pages {
			topicIDSet[p.TopicID] = struct{}{}
		}
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

	// ViewModelを構築
	backlinkMap := make(map[model.PageID]viewmodel.BacklinkList, len(backlinkPaginatedMap))
	for pageID, paginated := range backlinkPaginatedMap {
		var linkedPageNumber int32
		for _, p := range paginatedLinks.Pages {
			if p.ID == pageID {
				linkedPageNumber = int32(p.Number)
				break
			}
		}
		backlinkMap[pageID] = viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
			Pages:            paginated.Pages,
			TopicMap:         topicMap,
			Pagination:       viewmodel.NewPagination(1, paginated.TotalCount, int(viewmodel.BacklinkLimit)),
			SpaceIdentifier:  spaceIdentifier,
			PageNumber:       int32(pg.Number),
			LinkedPageNumber: linkedPageNumber,
		})
	}

	linkListVM := viewmodel.NewLinkList(viewmodel.NewLinkListInput{
		Pages:           paginatedLinks.Pages,
		TopicMap:        topicMap,
		BacklinkMap:     backlinkMap,
		Pagination:      viewmodel.NewPagination(int(currentPage), paginatedLinks.TotalCount, int(viewmodel.LinkLimit)),
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pg.Number),
	})

	// SSEフラグメントとしてリンク一覧を送信
	// カードをページネーションコンテナの前に追加し、コンテナ内のページネーション内容を更新する
	sse := datastar.NewSSE(w, r)

	if len(linkListVM.Items) > 0 {
		if err := sse.PatchElementTempl(components.LinkListCards(linkListVM), datastar.WithSelectorID("page-link-list-pagination"), datastar.WithModeBefore()); err != nil {
			slog.ErrorContext(ctx, "リンク一覧カードのSSE送信に失敗", "error", err)
			return
		}
	}

	if err := sse.PatchElementTempl(components.LinkListPagination(linkListVM), datastar.WithSelectorID("page-link-list-pagination"), datastar.WithModeInner()); err != nil {
		slog.ErrorContext(ctx, "リンク一覧ページネーションのSSE送信に失敗", "error", err)
		return
	}
}
