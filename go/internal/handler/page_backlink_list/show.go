package page_backlink_list

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

// Showはバックリンク一覧をHTMLフラグメントとして返します (GET /s/{space_identifier}/pages/{page_number}/links/{linked_page_number}/backlink_list)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// リンク先ページのバックリンク一覧は公開のページ表示画面に出る一覧の続きで、その
	// 「もっと見る」ボタンからゲストも到達する。何を返してよいかは閲覧者が開けるトピックから
	// UseCaseが判断する。
	user := middleware.UserFromContext(ctx)
	var userID *model.UserID
	if user != nil {
		userID = &user.ID
	}

	pageLinkContext := viewmodel.NormalizePageLinkContext(r.URL.Query().Get(viewmodel.PageLinkContextQueryParam))

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	pageNumberStr := chi.URLParam(r, "page_number")
	linkedPageNumberStr := chi.URLParam(r, "linked_page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	linkedPageNumber, err := strconv.ParseInt(linkedPageNumberStr, 10, 32)
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
	pageBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.PageBacklinkPageQueryParam, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// 本エンドポイントはパスが指すカードを進めるため、親ページは、リンク一覧が現在どこまで進んだか
	// ではなくそのカードのリンク一覧ページを指す。よってフラグメント側の名前で受け取る。編集画面は共有
	// 状態も送っており、そのlinked_page_parent_pageは画面が現在開いているカードのものだからである。
	// 値が無い場合は0のままにして、状態がリンク一覧自身のページへフォールバックできるようにする。
	// 親ページが存在しなかった頃のリクエストが意味していたのはその値である。
	parentLinkPage, ok := httppagination.ParseOptionalNumberParam(r, viewmodel.FragmentParentPageQueryParam)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// 応答はカードとその2つのページ番号を共有状態へ書き戻す。これにより、後続のフルページURLが
	// カードの実際に載るページを描画できる。
	linkState := viewmodel.PageLinkState{
		Context:              pageLinkContext,
		LinkPage:             linkPage,
		LinkedPageNumber:     int32(linkedPageNumber),
		LinkedBacklinkPage:   currentPage,
		LinkedPageParentPage: parentLinkPage,
		PageBacklinkPage:     pageBacklinkPage,
	}.Normalized()
	if !linkState.WithinCumulativeLimit(usecase.MaxCumulativeRelatedPagePages) {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// UseCaseを実行
	output, err := h.getBacklinkListUC.Execute(ctx, usecase.GetBacklinkListInput{
		SpaceIdentifier:  spaceIdentifier,
		PageNumber:       int32(pageNumber),
		LinkedPageNumber: int32(linkedPageNumber),
		UserID:           userID,
		CurrentPage:      currentPage,
		Limit:            viewmodel.BacklinkLimit,
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
		slog.ErrorContext(ctx, "バックリンクの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pagination, loadMoreCapped := viewmodel.NewRelatedPagePagination(currentPage, output.TotalCount, viewmodel.BacklinkLimit, linkState, linkState.CumulativePageLimit(usecase.MaxCumulativeRelatedPagePages))
	if int(currentPage) > pagination.Total {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	var linkedPageTitle string
	if output.LinkedPage.Title != nil {
		linkedPageTitle = *output.LinkedPage.Title
	}

	backlinkListVM := viewmodel.NewBacklinkList(viewmodel.NewBacklinkListInput{
		Pages:           output.Backlinks,
		TopicMap:        output.TopicMap,
		Pagination:      pagination,
		LoadMoreCapped:  loadMoreCapped,
		SpaceIdentifier: output.Space.Identifier,
		PageNumber:      int32(output.Page.Number),
		// 一覧は自分が描画されるページを自ら解決するため、親ページを運ばなかったリクエストでも、
		// カードが載るリンク一覧ページを指すリンクを組み立てられる。
		ParentLinkPage:   linkState.NestedBacklinkLinkPage(),
		LinkedPageNumber: int32(output.LinkedPage.Number),
		LinkedPageTitle:  linkedPageTitle,
		State:            linkState,
		// 各カードの編集リンクは、ページ表示画面の初回描画と同じく閲覧者自身の権限に従う。
		CanEdit: output.CanUpdatePage,
	})

	// HTMLフラグメントとしてバックリンク一覧を送信
	if err := components.BacklinkListResponse(backlinkListVM).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "バックリンク一覧のレンダリングに失敗", "error", err)
	}
}
