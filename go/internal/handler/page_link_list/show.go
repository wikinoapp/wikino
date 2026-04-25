package page_link_list

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show はリンク一覧の追加ページをHTMLフラグメントとして返します (GET /s/{space_identifier}/pages/{page_number}/link_list)
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
		handler.NotFound(w, r)
		return
	}

	// ページネーションパラメータを取得
	currentPage := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			currentPage = int32(p)
		}
	}

	// UseCaseを実行
	output, err := h.getLinkListUC.Execute(ctx, usecase.GetLinkListInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
		CurrentPage:     currentPage,
		LinkLimit:       viewmodel.LinkLimit,
		BacklinkLimit:   viewmodel.BacklinkLimit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "リンク一覧の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output == nil {
		handler.NotFound(w, r)
		return
	}

	// 認可チェック
	if !output.CanUpdatePage {
		handler.NotFound(w, r)
		return
	}

	// ViewModelを構築
	backlinkMap := make(map[model.PageID]viewmodel.BacklinkList, len(output.BacklinksPerPage))
	for pageID, backlinks := range output.BacklinksPerPage {
		var linkedPageNumber int32
		for _, p := range output.LinkedPages {
			if p.ID == pageID {
				linkedPageNumber = int32(p.Number)
				break
			}
		}
		backlinkMap[pageID] = viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
			Pages:            backlinks.Pages,
			TopicMap:         output.TopicMap,
			Pagination:       viewmodel.NewPagination(1, backlinks.TotalCount, int(viewmodel.BacklinkLimit)),
			SpaceIdentifier:  spaceIdentifier,
			PageNumber:       int32(output.Page.Number),
			LinkedPageNumber: linkedPageNumber,
		})
	}

	linkListVM := viewmodel.NewLinkList(viewmodel.NewLinkListInput{
		Pages:           output.LinkedPages,
		TopicMap:        output.TopicMap,
		BacklinkMap:     backlinkMap,
		Pagination:      viewmodel.NewPagination(int(currentPage), output.LinkedTotalCount, int(viewmodel.LinkLimit)),
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(output.Page.Number),
	})

	// HTMLフラグメントとしてリンク一覧を送信
	if err := components.LinkListResponse(linkListVM).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "リンク一覧のレンダリングに失敗", "error", err)
	}
}
