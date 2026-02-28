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

// Show はリンク一覧をSSEフラグメントとして返します (GET /go/s/{space_identifier}/pages/{page_number}/link_list)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := chi.URLParam(r, "space_identifier")
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

	const linkLimit int32 = 15
	const backlinkLimit int32 = 14

	// ページネーションパラメータを取得
	currentPage := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			currentPage = int32(p)
		}
	}

	var linkListVM viewmodel.LinkList
	if len(linkedPageIDs) > 0 {
		paginatedLinks, err := h.pageRepo.FindLinkedPagesPaginated(ctx, linkedPageIDs, space.ID, currentPage, linkLimit)
		if err != nil {
			slog.ErrorContext(ctx, "リンク先ページの取得に失敗", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		backlinkMap := make(map[model.PageID]viewmodel.BacklinkList, len(paginatedLinks.Pages))
		for _, linkedPage := range paginatedLinks.Pages {
			paginatedBacklinks, err := h.pageRepo.FindBacklinkedPagesPaginated(ctx, linkedPage.ID, space.ID, 1, backlinkLimit)
			if err != nil {
				slog.ErrorContext(ctx, "バックリンクの取得に失敗", "error", err, "page_id", linkedPage.ID)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			backlinkMap[linkedPage.ID] = viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
				Pages:      paginatedBacklinks.Pages,
				Pagination: viewmodel.NewPagination(1, paginatedBacklinks.TotalCount, int(backlinkLimit)),
			})
		}

		linkListVM = viewmodel.NewLinkList(viewmodel.NewLinkListInput{
			Pages:           paginatedLinks.Pages,
			BacklinkMap:     backlinkMap,
			Pagination:      viewmodel.NewPagination(int(currentPage), paginatedLinks.TotalCount, int(linkLimit)),
			SpaceIdentifier: spaceIdentifier,
			PageNumber:      int32(pg.Number),
		})
	}

	// SSEフラグメントとしてリンク一覧を送信
	sse := datastar.NewSSE(w, r)
	if err := sse.PatchElementTempl(components.LinkList(linkListVM), datastar.WithSelectorID("page-link-list"), datastar.WithModeInner()); err != nil {
		slog.ErrorContext(ctx, "リンク一覧のSSE送信に失敗", "error", err)
		return
	}
}
