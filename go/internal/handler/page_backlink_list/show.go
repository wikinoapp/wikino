package page_backlink_list

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

// Show はバックリンク一覧をHTMLフラグメントとして返します (GET /s/{space_identifier}/pages/{page_number}/links/{linked_page_number}/backlink_list)
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
		handler.NotFound(w, r)
		return
	}

	linkedPageNumber, err := strconv.ParseInt(linkedPageNumberStr, 10, 32)
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
	output, err := h.getBacklinkListUC.Execute(ctx, usecase.GetBacklinkListInput{
		SpaceIdentifier:  spaceIdentifier,
		PageNumber:       int32(pageNumber),
		LinkedPageNumber: int32(linkedPageNumber),
		UserID:           user.ID,
		CurrentPage:      currentPage,
		Limit:            viewmodel.BacklinkLimit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "バックリンクの取得に失敗", "error", err)
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

	backlinkListVM := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:            output.Backlinks,
		TopicMap:         output.TopicMap,
		Pagination:       viewmodel.NewPagination(int(currentPage), output.TotalCount, int(viewmodel.BacklinkLimit)),
		SpaceIdentifier:  spaceIdentifier,
		PageNumber:       int32(pageNumber),
		LinkedPageNumber: int32(linkedPageNumber),
	})

	// HTMLフラグメントとしてバックリンク一覧を送信
	if err := components.BacklinkList(backlinkListVM).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "バックリンク一覧のレンダリングに失敗", "error", err)
	}
}
