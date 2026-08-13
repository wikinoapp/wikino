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

// Show はバックリンク一覧をHTMLフラグメントとして返します (GET /s/{space_identifier}/pages/{page_number}/links/{linked_page_number}/backlink_list)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// The backlink list of a linked page is the continuation of a listing on the public page detail
	// screen, so a guest reaches it through its "load more" button. Which pages it may return is
	// decided by the usecase from the topics the viewer can open.
	//
	// [Ja] リンク先ページのバックリンク一覧は公開のページ表示画面に出る一覧の続きで、その
	// 「もっと見る」ボタンからゲストも到達する。何を返してよいかは閲覧者が開けるトピックから
	// UseCase が判断する。
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

	// Parse the pagination parameter. A page whose SQL offset cannot fit the query's int32
	// parameter is rejected before invoking the usecase.
	//
	// [Ja] ページネーションパラメータを取得する。SQL offset がクエリの int32 パラメータに収まらない
	// ページは UseCase 呼び出し前に拒否する。
	currentPage, ok := httppagination.ParsePageParam(r, viewmodel.BacklinkLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}
	// The other listings' pages ride along so that the links this fragment renders keep pointing at
	// the state the whole screen is in, instead of resetting them to their first page.
	//
	// [Ja] 他の一覧のページを一緒に受け取ることで、このフラグメントが描画するリンクが画面全体の状態を
	// 指し続けるようにする。受け取らないと、他の一覧が 1 ページ目へ戻ってしまう。
	linkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkPageQueryParam, viewmodel.LinkLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}
	pageBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.PageBacklinkPageQueryParam, viewmodel.PageBacklinkLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// This endpoint advances the card named by the path, so the parent page names that card's
	// link-list page rather than whichever page the link list has since reached. It therefore
	// travels under the fragment-scoped name: the editor also sends the shared state, whose
	// linked_page_parent_page belongs to the card the screen currently holds open. An absent value
	// stays zero so the state falls back to the link list's own page, which is what a request built
	// before the parent page existed meant.
	//
	// [Ja] 本エンドポイントはパスが指すカードを進めるため、親ページは、リンク一覧が現在どこまで進んだか
	// ではなくそのカードのリンク一覧ページを指す。よってフラグメント側の名前で受け取る。編集画面は共有
	// 状態も送っており、その linked_page_parent_page は画面が現在開いているカードのものだからである。
	// 値が無い場合は 0 のままにして、状態がリンク一覧自身のページへフォールバックできるようにする。
	// 親ページが存在しなかった頃のリクエストが意味していたのはその値である。
	parentLinkPage, ok := httppagination.ParseOptionalNumberParam(r, viewmodel.FragmentParentPageQueryParam)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// The response writes the card and its two page numbers back into the shared state, which is
	// what lets a later full-page URL render the page the card is actually on.
	//
	// [Ja] 応答はカードとその 2 つのページ番号を共有状態へ書き戻す。これにより、後続のフルページ URL が
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
		// The listing resolves the page it renders itself on, so a request that carried no parent
		// page still builds links naming the link-list page the card is on.
		//
		// [Ja] 一覧は自分が描画されるページを自ら解決するため、親ページを運ばなかったリクエストでも、
		// カードが載るリンク一覧ページを指すリンクを組み立てられる。
		ParentLinkPage:   linkState.NestedBacklinkLinkPage(),
		LinkedPageNumber: int32(output.LinkedPage.Number),
		LinkedPageTitle:  linkedPageTitle,
		State:            linkState,
		// The per-card edit link follows the viewer's own permission, the same way the initial
		// listing on the page detail screen does.
		//
		// [Ja] 各カードの編集リンクは、ページ表示画面の初回描画と同じく閲覧者自身の権限に従う。
		CanEdit: output.CanUpdatePage,
	})

	// HTMLフラグメントとしてバックリンク一覧を送信
	if err := components.BacklinkListResponse(backlinkListVM).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "バックリンク一覧のレンダリングに失敗", "error", err)
	}
}
