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

// Showはページレベルのバックリンク一覧をHTMLフラグメントとして返します (GET /s/{space_identifier}/pages/{page_number}/backlinks)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// バックリンク一覧は公開のページ表示画面に出る一覧の続きで、その「もっと見る」ボタンから
	// ゲストも到達する。何を返してよいかは閲覧者が開けるトピックからUseCaseが判断する。
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

	// ページネーションパラメータを取得する。SQL offsetがクエリのint32パラメータに収まらない
	// ページはUseCase呼び出し前に拒否する。
	currentPage, ok := httppagination.ParsePageParam(r, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// 他の一覧のページを一緒に受け取ることで、このフラグメントが描画するリンクが画面全体の状態を
	// 指し続けるようにする。受け取らないと、他の一覧が1ページ目へ戻ってしまう。
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
	// 選択カードのリンク一覧ページは、そのカードを指す2つの値と一緒に受け取る。本応答が描画する
	// フルページフォールバックが、カードを含むページを指す必要があるためである。値が無い場合は0のままに
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
		// 各カードの編集リンクは、ページ表示画面の初回描画と同じく閲覧者自身の権限に従う。
		CanEdit: output.CanUpdatePage,
	})

	// HTMLフラグメントとしてバックリンク一覧を送信
	if err := components.PageBacklinkListResponse(backlinkListVM).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "バックリンク一覧のレンダリングに失敗", "error", err)
	}
}
