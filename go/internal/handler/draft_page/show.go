package draft_page

import (
	"log/slog"
	"math"
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

// Show returns the draft's save time and the two related-page listings as an HTML fragment of
// out-of-band swaps, which the editor requests after every draft autosave
// (GET /s/{space_identifier}/pages/{page_number}/draft_page).
//
// The related-page state of the screen travels in the query string, so the swapped-in listings
// render the same slices the reader is looking at and their links keep pointing at that state.
//
// [Ja] Show は下書きの保存時刻と 2 つの関連ページ一覧を、OOB スワップの HTML フラグメントとして返す
// (GET /s/{space_identifier}/pages/{page_number}/draft_page)。編集画面が下書きの自動保存ごとに
// 要求する。
//
// 画面の関連ページ状態はクエリ文字列で受け取る。差し替わる一覧が閲覧者の見ている範囲を描画し、
// その中のリンクもその状態を指し続けるようにするためである。
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
		handler.NotFound(w, r)
		return
	}

	// 認可チェック
	if !output.CanUpdatePage {
		handler.NotFound(w, r)
		return
	}

	// This endpoint re-renders all three related-page listings at once, so every page number arrives
	// under its own listing's name rather than under the conventional "page" of the single-listing
	// fragment endpoints. That is also what lets the editor send them straight from the shared state
	// element, whose inputs carry these same names.
	//
	// [Ja] 本エンドポイントは 3 つの関連ページ一覧をまとめて描画し直すため、各ページ番号は 1 つの一覧
	// だけを返すフラグメントエンドポイントの慣例的な "page" ではなく、それぞれの一覧の名前で受け取る。
	// これにより、同じ名前を入力に持つ共有状態の要素から編集画面がそのまま送れるようにもなる。
	pageLinkContext := viewmodel.NormalizePageLinkContext(r.URL.Query().Get(viewmodel.PageLinkContextQueryParam))
	if !pageLinkContext.IsEdit() {
		handler.NotFound(w, r)
		return
	}

	currentPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkPageQueryParam, viewmodel.LinkLimit)
	if !ok {
		handler.NotFound(w, r)
		return
	}
	linkedBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkedBacklinkPageQueryParam, viewmodel.BacklinkLimit)
	if !ok {
		handler.NotFound(w, r)
		return
	}
	pageBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.PageBacklinkPageQueryParam, viewmodel.PageBacklinkLimit)
	if !ok {
		handler.NotFound(w, r)
		return
	}
	linkedPageNumber, ok := httppagination.ParseOptionalNumberParam(r, viewmodel.LinkedPageNumberQueryParam)
	if !ok {
		handler.NotFound(w, r)
		return
	}
	linkedPageParentPage, ok := httppagination.ParseOptionalNumberParam(r, viewmodel.LinkedPageParentPageQueryParam)
	if !ok {
		handler.NotFound(w, r)
		return
	}

	linkState := viewmodel.PageLinkState{
		Context:              pageLinkContext,
		LinkPage:             currentPage,
		LinkedPageNumber:     linkedPageNumber,
		LinkedBacklinkPage:   linkedBacklinkPage,
		LinkedPageParentPage: linkedPageParentPage,
		PageBacklinkPage:     pageBacklinkPage,
	}.Normalized()
	if !linkState.WithinCumulativeLimit(usecase.MaxCumulativeRelatedPagePages) {
		handler.NotFound(w, r)
		return
	}

	// The cumulative editor rebuilds its loaded prefix, while the paginated editor replaces exactly
	// one requested page. Both paths reconcile stale state after the draft changes. A paginated request
	// whose page disappeared must run once more at the clamped page because its first fixed-size slice
	// is empty; the cumulative response already contains every surviving page and needs no refetch.
	//
	// [Ja] 累積編集画面は読み込み済みの先頭範囲を再構築し、ページ単位の編集画面は要求された 1 ページ
	// だけを差し替える。どちらも下書き変更後に古い状態を整合させる。ページ単位の要求ページが消えた場合、
	// 最初の固定長スライスは空なので、丸めたページでもう一度取得する。累積応答は残存ページをすべて含むため
	// 再取得不要である。
	includePrecedingPages := linkState.Context.IncludesPrecedingPages()
	fetchLinkData := func(state viewmodel.PageLinkState) (*usecase.GetEditLinkDataOutput, error) {
		return h.getEditLinkDataUC.Execute(ctx, usecase.GetEditLinkDataInput{
			Page:                   output.Page,
			DraftPage:              output.DraftPage,
			SpaceID:                output.Space.ID,
			CurrentPage:            state.LinkPage,
			LinkLimit:              viewmodel.LinkLimit,
			BacklinkLimit:          viewmodel.BacklinkLimit,
			PageBacklinkLimit:      viewmodel.PageBacklinkLimit,
			LinkedPageNumber:       state.LinkedPageNumber,
			LinkedPageBacklinkPage: state.LinkedBacklinkPage,
			PageBacklinkPage:       state.PageBacklinkPage,
			IncludePrecedingPages:  includePrecedingPages,
		})
	}

	linkData, err := fetchLinkData(linkState)
	if err != nil {
		slog.ErrorContext(ctx, "リンクデータの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !includePrecedingPages {
		topLevelState := reconcileTopLevelRelatedPageState(linkState, linkData)
		if topLevelState.LinkPage != linkState.LinkPage ||
			topLevelState.PageBacklinkPage != linkState.PageBacklinkPage {
			linkState = topLevelState
			linkData, err = fetchLinkData(linkState)
			if err != nil {
				slog.ErrorContext(ctx, "最上位一覧の整合後のリンクデータ取得に失敗", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
	}

	reconciledState := reconcileRelatedPageState(linkState, linkData)
	if !includePrecedingPages && reconciledState != linkState {
		linkState = reconciledState
		linkData, err = fetchLinkData(linkState)
		if err != nil {
			slog.ErrorContext(ctx, "ネスト一覧の整合後のリンクデータ取得に失敗", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		reconciledState = reconcileRelatedPageState(linkState, linkData)
	}
	linkState = reconciledState

	// ViewModelを構築
	backlinksPerPage := make(map[model.PageID]*viewmodel.PageSliceWithCount, len(linkData.BacklinksPerPage))
	for pageID, backlinks := range linkData.BacklinksPerPage {
		backlinksPerPage[pageID] = &viewmodel.PageSliceWithCount{
			Pages:       backlinks.Pages,
			TotalCount:  backlinks.TotalCount,
			CurrentPage: linkState.BacklinkPageFor(linkData.LinkedPages, pageID),
		}
	}

	// The draft fragment is only reachable by a member editing the page, so the listed cards keep
	// their edit links.
	//
	// [Ja] 下書きフラグメントはページを編集中のメンバーしか到達しないため、一覧のカードは編集リンクを
	// 出したままにする。
	editLinkData := viewmodel.BuildPageLinkData(viewmodel.BuildPageLinkDataInput{
		LinkedPages:         linkData.LinkedPages,
		LinkedTotalCount:    linkData.LinkedTotalCount,
		BacklinksPerPage:    backlinksPerPage,
		PageBacklinks:       linkData.PageBacklinks,
		PageBacklinkCount:   linkData.PageBacklinkCount,
		Topics:              linkData.LinkTopics,
		SpaceIdentifier:     spaceIdentifier,
		PageNumber:          int32(output.Page.Number),
		LinkedPageFirstPage: linkedPageFirstPage(linkState),
		CumulativePageLimit: linkState.CumulativePageLimit(usecase.MaxCumulativeRelatedPagePages),
		State:               linkState,
		CanEdit:             true,
	})

	// OOBスワップ付きHTMLフラグメントをレンダリング
	responseData := components.DraftPageShowResponseData{
		LinkList:     editLinkData.LinkList,
		BacklinkList: editLinkData.BacklinkList,
		State:        linkState,
	}
	if output.DraftPage != nil {
		responseData.HasDraft = true
		responseData.ModifiedAt = output.DraftPage.ModifiedAt
	}

	if err := components.DraftPageShowResponse(responseData).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "下書きページレスポンスのレンダリングに失敗", "error", err)
	}
}

// reconcileRelatedPageState moves every editor listing page into the range that still exists after
// a draft change. A nested state survives only while its parent card remains in the fetched link
// range; otherwise both halves reset together.
//
// [Ja] reconcileRelatedPageState は、下書き変更後も存在する範囲へ編集画面の各一覧ページを戻す。
// ネスト状態は親カードが取得済みリンク範囲に残る間だけ維持し、消えた場合は両方をまとめてリセットする。
func reconcileRelatedPageState(state viewmodel.PageLinkState, data *usecase.GetEditLinkDataOutput) viewmodel.PageLinkState {
	state = reconcileTopLevelRelatedPageState(state, data)

	if state.LinkedPageNumber <= 0 {
		state.LinkedBacklinkPage = 1
		state.LinkedPageParentPage = 0
		return state
	}

	for index, linkedPage := range data.LinkedPages {
		if int32(linkedPage.Number) != state.LinkedPageNumber {
			continue
		}

		backlinks := data.BacklinksPerPage[linkedPage.ID]
		if backlinks == nil {
			state.LinkedPageNumber = 0
			state.LinkedBacklinkPage = 1
			state.LinkedPageParentPage = 0
			return state
		}

		state.LinkedBacklinkPage = relatedPageAtOrBeforeLast(state.LinkedBacklinkPage, backlinks.TotalCount, viewmodel.BacklinkLimit)
		// A draft change can move the card into a different link-list page, so the parent page is
		// recomputed from where the card sits now rather than carried over. The same rule as the
		// listing itself uses keeps the state and the rendered cards naming the same page.
		//
		// [Ja] 下書きの変更でカードが別のリンク一覧ページへ移ることがあるため、親ページは引き継がず、
		// 現在の位置から求め直す。一覧自身と同じ規則を使うことで、状態と描画したカードが同じページを
		// 指し続ける。
		state.LinkedPageParentPage = linkedPageFirstPage(state) + int32(index)/viewmodel.LinkLimit
		return state
	}

	state.LinkedPageNumber = 0
	state.LinkedBacklinkPage = 1
	state.LinkedPageParentPage = 0
	return state
}

// linkedPageFirstPage returns the one-based link-list page of the first card the response renders.
// A cumulative refresh rebuilds the listing from its first page, while a one-page response starts at
// the page it was asked for.
//
// [Ja] linkedPageFirstPage は、応答が描画する先頭カードのリンク一覧ページ (1 始まり) を返す。累積
// 再取得は 1 ページ目から一覧を組み立て直し、1 ページ分の応答は要求されたページから始まる。
func linkedPageFirstPage(state viewmodel.PageLinkState) int32 {
	if state.Context.IncludesPrecedingPages() {
		return 1
	}

	return state.LinkPage
}

// reconcileTopLevelRelatedPageState clamps the two screen-wide listings without touching the
// selected nested card. The paginated editor uses it before refetching a moved final link page, so
// nested state is evaluated against the surviving parent cards rather than an obsolete empty slice.
//
// [Ja] reconcileTopLevelRelatedPageState は、選択中のネストカードに触れず画面全体の 2 一覧を丸める。
// ページ単位編集画面は移動したリンク最終ページを再取得する前にこれを使い、古い空スライスではなく
// 残存する親カードに対してネスト状態を判定する。
func reconcileTopLevelRelatedPageState(state viewmodel.PageLinkState, data *usecase.GetEditLinkDataOutput) viewmodel.PageLinkState {
	state = state.Normalized()
	state.LinkPage = relatedPageAtOrBeforeLast(state.LinkPage, data.LinkedTotalCount, viewmodel.LinkLimit)
	state.PageBacklinkPage = relatedPageAtOrBeforeLast(state.PageBacklinkPage, data.PageBacklinkCount, viewmodel.PageBacklinkLimit)
	return state
}

func relatedPageAtOrBeforeLast(page int32, totalCount int64, perPage int32) int32 {
	if page <= 1 || totalCount <= 0 || perPage <= 0 {
		return 1
	}

	lastPage := (totalCount + int64(perPage) - 1) / int64(perPage)
	if int64(page) > lastPage {
		if lastPage > math.MaxInt32 {
			return page
		}

		// lastPage is positive and below both page and math.MaxInt32 in this branch.
		// [Ja] この分岐の lastPage は正で、page と math.MaxInt32 の双方より小さい。
		// #nosec G115 -- bounds are proven immediately above.
		return int32(lastPage)
	}

	return page
}
