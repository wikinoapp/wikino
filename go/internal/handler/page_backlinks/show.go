package page_backlinks

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/httppagination"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Show はページレベルのバックリンク一覧をHTMLフラグメントとして返します (GET /s/{space_identifier}/pages/{page_number}/backlinks)
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

	// Parse the pagination parameter. A page whose SQL offset cannot fit the query's int32
	// parameter is rejected before invoking the usecase.
	//
	// [Ja] ページネーションパラメータを取得する。SQL offset がクエリの int32 パラメータに収まらない
	// ページは UseCase 呼び出し前に拒否する。
	currentPage, ok := httppagination.ParsePageParam(r, viewmodel.PageBacklinkLimit)
	if !ok {
		handler.NotFound(w, r)
		return
	}

	// UseCaseを実行
	output, err := h.getPageBacklinksUC.Execute(ctx, usecase.GetPageBacklinksInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          &user.ID,
		CurrentPage:     currentPage,
		Limit:           viewmodel.PageBacklinkLimit,
	})
	if err != nil {
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				handler.NotFound(w, r)
			default:
				slog.ErrorContext(ctx, ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.ErrorContext(ctx, "ページレベルのバックリンクの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 認可チェック
	if !output.CanUpdatePage {
		handler.NotFound(w, r)
		return
	}

	backlinkListVM := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:           output.Backlinks,
		TopicMap:        output.TopicMap,
		Pagination:      viewmodel.NewPagination(int(currentPage), output.TotalCount, int(viewmodel.PageBacklinkLimit)),
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(output.Page.Number),
	})

	// HTMLフラグメントとしてバックリンク一覧を送信
	if err := components.PageBacklinkListResponse(backlinkListVM).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "バックリンク一覧のレンダリングに失敗", "error", err)
	}
}
