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

	// The backlink list is the continuation of the listing on the public page detail screen, so a
	// guest reaches it through its "load more" button. Which pages it may return is decided by the
	// usecase from the topics the viewer can open.
	//
	// [Ja] バックリンク一覧は公開のページ表示画面に出る一覧の続きで、その「もっと見る」ボタンから
	// ゲストも到達する。何を返してよいかは閲覧者が開けるトピックから UseCase が判断する。
	user := middleware.UserFromContext(ctx)
	var userID *model.UserID
	if user != nil {
		userID = &user.ID
	}

	pageLinkContext := viewmodel.NormalizePageLinkContext(r.URL.Query().Get(viewmodel.PageLinkContextQueryParam))

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	pageNumberStr := chi.URLParam(r, "page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// Parse the pagination parameter. A page whose SQL offset cannot fit the query's int32
	// parameter is rejected before invoking the usecase.
	//
	// [Ja] ページネーションパラメータを取得する。SQL offset がクエリの int32 パラメータに収まらない
	// ページは UseCase 呼び出し前に拒否する。
	currentPage, ok := httppagination.ParsePageParam(r, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// The other listings' pages ride along so that the links this fragment renders keep pointing at
	// the state the whole screen is in, instead of resetting them to their first page.
	//
	// [Ja] 他の一覧のページを一緒に受け取ることで、このフラグメントが描画するリンクが画面全体の状態を
	// 指し続けるようにする。受け取らないと、他の一覧が 1 ページ目へ戻ってしまう。
	linkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkPageQueryParam, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}
	linkedBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkedBacklinkPageQueryParam, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}
	linkedPageNumber, ok := httppagination.ParseOptionalNumberParam(r, viewmodel.LinkedPageNumberQueryParam)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}
	// The selected card's own link-list page rides along with the two halves that name it, because
	// the full-page fallback this response renders has to point at the page holding that card. An
	// absent value stays zero so the state falls back to the link list's own page.
	//
	// [Ja] 選択カードのリンク一覧ページは、そのカードを指す 2 つの値と一緒に受け取る。本応答が描画する
	// フルページフォールバックが、カードを含むページを指す必要があるためである。値が無い場合は 0 のままに
	// して、状態がリンク一覧自身のページへフォールバックできるようにする。
	linkedPageParentPage, ok := httppagination.ParseOptionalNumberParam(r, viewmodel.LinkedPageParentPageQueryParam)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	linkState := viewmodel.PageLinkState{
		Context:              pageLinkContext,
		LinkPage:             linkPage,
		LinkedPageNumber:     linkedPageNumber,
		LinkedBacklinkPage:   linkedBacklinkPage,
		LinkedPageParentPage: linkedPageParentPage,
		PageBacklinkPage:     currentPage,
	}.Normalized()
	if !linkState.WithinCumulativeLimit(usecase.MaxCumulativeRelatedPagePages) {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// UseCaseを実行
	output, err := h.getPageBacklinksUC.Execute(ctx, usecase.GetPageBacklinksInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          userID,
		CurrentPage:     currentPage,
		Limit:           viewmodel.PageBacklinkLimit,
	})
	if err != nil {
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				handler.RelatedPageListNotFound(w, r)
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

	pagination, loadMoreCapped := viewmodel.NewRelatedPagePagination(currentPage, output.TotalCount, viewmodel.PageBacklinkLimit, linkState, linkState.CumulativePageLimit(usecase.MaxCumulativeRelatedPagePages))
	if int(currentPage) > pagination.Total {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	backlinkListVM := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:           output.Backlinks,
		TopicMap:        output.TopicMap,
		Pagination:      pagination,
		LoadMoreCapped:  loadMoreCapped,
		SpaceIdentifier: output.Space.Identifier,
		PageNumber:      int32(output.Page.Number),
		State:           linkState,
		// The per-card edit link follows the viewer's own permission, the same way the initial
		// listing on the page detail screen does.
		//
		// [Ja] 各カードの編集リンクは、ページ表示画面の初回描画と同じく閲覧者自身の権限に従う。
		CanEdit: output.CanUpdatePage,
	})

	// HTMLフラグメントとしてバックリンク一覧を送信
	if err := components.PageBacklinkListResponse(backlinkListVM).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "バックリンク一覧のレンダリングに失敗", "error", err)
	}
}
