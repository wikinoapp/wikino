package page_link_list

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

// Showはリンク一覧の追加ページをHTMLフラグメントとして返します (GET /s/{space_identifier}/pages/{page_number}/link_list)
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// リンク一覧は公開のページ表示画面に出る一覧の続きで、その「もっと見る」ボタンからゲストも
	// 到達する。何を返してよいかは閲覧者が開けるトピックからUseCaseが判断する。
	user := middleware.UserFromContext(ctx)
	var userID *model.UserID
	if user != nil {
		userID = &user.ID
	}

	pageLinkContext := viewmodel.NormalizePageLinkContext(r.URL.Query().Get(viewmodel.PageLinkContextQueryParam))
	linkListSource := usecase.LinkListSourceDraft
	if pageLinkContext == viewmodel.PageLinkContextShow {
		linkListSource = usecase.LinkListSourceSaved
	}

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
	linkedBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkedBacklinkPageQueryParam, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}
	pageBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.PageBacklinkPageQueryParam, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}
	linkedPageNumber, ok := httppagination.ParseOptionalNumberParam(r, viewmodel.LinkedPageNumberQueryParam)
	if !ok {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	linkState := viewmodel.PageLinkState{
		Context:            pageLinkContext,
		LinkPage:           currentPage,
		LinkedPageNumber:   linkedPageNumber,
		LinkedBacklinkPage: linkedBacklinkPage,
		PageBacklinkPage:   pageBacklinkPage,
	}.Normalized()
	if !linkState.WithinCumulativeLimit(usecase.MaxCumulativeRelatedPagePages) {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// UseCaseを実行
	output, err := h.getLinkListUC.Execute(ctx, usecase.GetLinkListInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          userID,
		CurrentPage:     currentPage,
		LinkLimit:       viewmodel.LinkLimit,
		BacklinkLimit:   viewmodel.BacklinkLimit,
		Source:          linkListSource,
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
		slog.ErrorContext(ctx, "リンク一覧の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pagination, loadMoreCapped := viewmodel.NewRelatedPagePagination(currentPage, output.LinkedTotalCount, viewmodel.LinkLimit, linkState, linkState.CumulativePageLimit(usecase.MaxCumulativeRelatedPagePages))
	if int(currentPage) > pagination.Total {
		handler.RelatedPageListNotFound(w, r)
		return
	}

	// ViewModelを構築
	backlinksPerPage := make(map[model.PageID]*viewmodel.PageSliceWithCount, len(output.BacklinksPerPage))
	for pageID, backlinks := range output.BacklinksPerPage {
		backlinksPerPage[pageID] = &viewmodel.PageSliceWithCount{
			Pages:      backlinks.Pages,
			TotalCount: backlinks.TotalCount,
		}
	}

	// 本フラグメントのカードはいずれも画面に新しく加わるため、ネストしたバックリンク一覧は
	// それぞれ1ページ目から始まる。
	backlinkMap := viewmodel.NewLinkedPageBacklinkLists(viewmodel.NewLinkedPageBacklinkListsInput{
		LinkedPages:         output.LinkedPages,
		BacklinksPerPage:    backlinksPerPage,
		TopicMap:            output.TopicMap,
		SpaceIdentifier:     output.Space.Identifier,
		PageNumber:          int32(output.Page.Number),
		LinkedPageFirstPage: currentPage,
		CumulativePageLimit: linkState.CumulativePageLimit(usecase.MaxCumulativeRelatedPagePages),
		State:               linkState,
		CanEdit:             output.CanUpdatePage,
	})

	linkListVM := viewmodel.NewLinkList(viewmodel.NewLinkListInput{
		Pages:           output.LinkedPages,
		TopicMap:        output.TopicMap,
		BacklinkMap:     backlinkMap,
		Pagination:      pagination,
		LoadMoreCapped:  loadMoreCapped,
		SpaceIdentifier: output.Space.Identifier,
		PageNumber:      int32(output.Page.Number),
		State:           linkState,
		// 各カードの編集リンクは、ページ表示画面の初回描画と同じく閲覧者自身の権限に従う。
		CanEdit: output.CanUpdatePage,
	})

	// HTMLフラグメントとしてリンク一覧を送信
	if err := components.LinkListResponse(linkListVM).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "リンク一覧のレンダリングに失敗", "error", err)
	}
}
