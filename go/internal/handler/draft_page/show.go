package draft_page

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
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show は下書き保存時刻とリンク一覧をSSEフラグメントとして返します (GET /s/{space_identifier}/pages/{page_number}/draft_page)
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

	// UseCaseでデータを取得
	output, err := h.getPageDetailUC.Execute(ctx, usecase.GetPageDetailInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "ページ詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		http.NotFound(w, r)
		return
	}

	// 認可チェック
	topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
	if !topicPolicy.CanUpdatePage(output.Page) {
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

	// 編集画面用のリンクデータを取得
	linkData, err := h.getEditLinkDataUC.Execute(ctx, usecase.GetEditLinkDataInput{
		Page:              output.Page,
		DraftPage:         output.DraftPage,
		SpaceID:           output.Space.ID,
		CurrentPage:       currentPage,
		LinkLimit:         viewmodel.LinkLimit,
		BacklinkLimit:     viewmodel.BacklinkLimit,
		PageBacklinkLimit: viewmodel.PageBacklinkLimit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "リンクデータの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ViewModelを構築
	backlinksPerPage := make(map[model.PageID]*viewmodel.PageSliceWithCount, len(linkData.BacklinksPerPage))
	for pageID, backlinks := range linkData.BacklinksPerPage {
		backlinksPerPage[pageID] = &viewmodel.PageSliceWithCount{
			Pages:      backlinks.Pages,
			TotalCount: backlinks.TotalCount,
		}
	}

	editLinkData := viewmodel.BuildEditLinkData(viewmodel.BuildEditLinkDataInput{
		LinkedPages:       linkData.LinkedPages,
		LinkedTotalCount:  linkData.LinkedTotalCount,
		BacklinksPerPage:  backlinksPerPage,
		PageBacklinks:     linkData.PageBacklinks,
		PageBacklinkCount: linkData.PageBacklinkCount,
		Topics:            linkData.LinkTopics,
		SpaceIdentifier:   spaceIdentifier,
		PageNumber:        int32(output.Page.Number),
		CurrentPage:       currentPage,
	})
	linkListVM := editLinkData.LinkList
	backlinkListVM := editLinkData.BacklinkList

	// SSEフラグメントを送信
	sse := datastar.NewSSE(w, r)

	// 保存時刻フラグメントを送信（下書きが存在する場合のみ）
	if output.DraftPage != nil {
		if err := sse.PatchElementTempl(components.DraftSavedTime(output.DraftPage.ModifiedAt, user.TimeZone), datastar.WithSelectorID("page-draft-saved-at"), datastar.WithModeOuter()); err != nil {
			slog.ErrorContext(ctx, "保存時刻のSSE送信に失敗", "error", err)
			return
		}
	}

	// リンク一覧フラグメントを送信
	if err := sse.PatchElementTempl(components.LinkList(linkListVM), datastar.WithSelectorID("page-link-list"), datastar.WithModeInner()); err != nil {
		slog.ErrorContext(ctx, "リンク一覧のSSE送信に失敗", "error", err)
		return
	}

	// バックリンク一覧フラグメントを送信
	if err := sse.PatchElementTempl(components.PageBacklinkList(backlinkListVM), datastar.WithSelectorID("page-backlink-list"), datastar.WithModeInner()); err != nil {
		slog.ErrorContext(ctx, "バックリンク一覧のSSE送信に失敗", "error", err)
		return
	}
}
