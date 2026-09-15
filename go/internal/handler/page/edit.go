package page

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/httppagination"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// zenModeCookieNameはエディタのZenモード状態を保持するクッキー ("1" でON。OFFは
// クッキーなし)。web/zen-mode.tsが書き込むためHttpOnlyではない。名前は同スクリプトと
// 同期させること。
const zenModeCookieName = "wikino_zen_mode"

// zenModeFromRequestはリクエストのクッキーからZenモード状態を読み取ります。
func zenModeFromRequest(r *http.Request) bool {
	cookie, err := r.Cookie(zenModeCookieName)
	return err == nil && cookie.Value == "1"
}

// Editはページ編集フォームを表示します (GET /s/{space_identifier}/pages/{page_number}/edit)
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
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

	pageLinkContext := viewmodel.NormalizePageLinkContext(r.URL.Query().Get(viewmodel.PageLinkContextQueryParam))
	if !pageLinkContext.IsEdit() {
		handler.NotFound(w, r)
		return
	}
	linkState, ok := parseRelatedPageState(r, pageLinkContext)
	if !ok {
		handler.NotFound(w, r)
		return
	}

	// UseCaseでデータを取得
	output, err := h.getPageDetailUC.Execute(ctx, usecase.GetPageDetailInput{
		SpaceIdentifier:       spaceIdentifier,
		PageNumber:            int32(pageNumber),
		UserID:                user.ID,
		IncludeDraftPages:     true,
		IncludeDraftRevisions: true,
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

	// 編集画面用のリンクデータを取得
	linkData, err := h.getEditLinkDataUC.Execute(ctx, usecase.GetEditLinkDataInput{
		Page:                   output.Page,
		DraftPage:              output.DraftPage,
		SpaceID:                output.Space.ID,
		CurrentPage:            linkState.LinkPage,
		LinkLimit:              viewmodel.LinkLimit,
		BacklinkLimit:          viewmodel.BacklinkLimit,
		PageBacklinkLimit:      viewmodel.PageBacklinkLimit,
		LinkedPageNumber:       linkState.LinkedPageNumber,
		LinkedPageBacklinkPage: linkState.LinkedBacklinkPage,
		PageBacklinkPage:       linkState.PageBacklinkPage,
	})
	if err != nil {
		slog.ErrorContext(ctx, "リンクデータの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !relatedPageStateInRange(linkState, relatedPageCounts{
		LinkedTotalCount:      linkData.LinkedTotalCount,
		PageBacklinkCount:     linkData.PageBacklinkCount,
		LinkedPages:           linkData.LinkedPages,
		BacklinkCountByPageID: linkedPageBacklinkCounts(linkData.BacklinksPerPage),
	}) {
		handler.NotFound(w, r)
		return
	}

	linkResult := buildEditLinkResult(buildEditLinkResultInput{
		LinkData:        linkData,
		SpaceIdentifier: output.Space.Identifier,
		Page:            output.Page,
		State:           linkState,
	})

	// 編集画面用のページViewModelを生成
	pageVM := viewmodel.NewPageForEdit(output.Page, output.DraftPage)
	spaceVM := viewmodel.NewSpace(output.Space)
	topicVM := viewmodel.NewTopic(output.Topic)

	// URLではなく保存済みの識別子からリンクを組み立て、画面内のリンクの表記を揃える。
	spaceIdentVM := spaceVM.Identifier

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "page_edit_title", map[string]any{
		"SpaceName": output.Space.Name,
	})
	meta.CurrentSpaceIdentifier = spaceIdentVM

	manualSaveURL := string(templates.PageDraftPageRevisionPath(spaceIdentVM, int32(output.Page.Number)))

	editData := pagepages.EditPageData{
		CSRFToken:               csrfToken,
		Page:                    pageVM,
		Space:                   spaceVM,
		Topic:                   topicVM,
		LinkList:                linkResult.LinkList,
		BacklinkList:            linkResult.BacklinkList,
		RelatedPageState:        linkState,
		ManualSaveURL:           manualSaveURL,
		CreateSuggestionSaveURL: manualSaveURL + "?redirect_to=suggestion_new",
		// 編集画面は同一スペース内のため、各下書きカードのスペースラベルを省く。
		DraftPages:     viewmodel.NewCardLinkDraftPagesWithoutSpace(output.DraftPages),
		DraftRevisions: viewmodel.NewDraftPageRevisions(output.DraftPageRevisions, output.DraftPageRevisionTotalCount),
		ZenMode:        zenModeFromRequest(r),
	}

	if output.Suggestion != nil && output.DraftPage != nil && output.DraftPage.SuggestionPageID != nil {
		editData.SuggestionNumber = int32(output.Suggestion.Number)
		editData.SuggestionURL = string(templates.SuggestionPagePath(spaceIdentVM, int32(output.Suggestion.Number), string(*output.DraftPage.SuggestionPageID)))
		editData.SuggestionShowURL = string(templates.SuggestionShowPath(spaceIdentVM, int32(output.Suggestion.Number)))
	}

	content := pagepages.Edit(editData)

	// 編集画面はグローバルナビの状態をGlobalNavで供給する。PageNamePageEditはどのナビ項目にも
	// 一致しないため、いずれの項目もアクティブにならない (画面内のナビゲーションはナビではなく
	// 下書き一覧カラムが担う)。
	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		HideFooter: true,
		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNamePageEdit,
			SignedIn:        true,
			UserAtname:      user.Atname,
			SpaceIdentifier: spaceIdentVM,
		},

		BreadcrumbHeader: editBreadcrumbHeaderData(ctx, spaceVM, topicVM),
	}

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// editBreadcrumbMaxWidthClassは編集画面の広い本文幅にパンくずを揃える。
const editBreadcrumbMaxWidthClass = "max-w-6xl"

// editBreadcrumbHeaderDataは編集画面のパンくずヘッダーを組み立てる。EditとUpdate
// (バリデーションエラー時に編集画面を再描画する) の両方が表示する。編集画面が現在地のため、経路は
// aria-currentを持つリンク無しの項目で締める。ラベルは編集対象のページではなく画面自身を表す。
// 編集画面は自身の見出しを持たず、閲覧者がまだタイトルを付けていない下書きでは項目が空になって
// しまうためである。どちらの画面も認証必須のため、経路は常にホームから始める。
func editBreadcrumbHeaderData(ctx context.Context, space viewmodel.Space, topic viewmodel.Topic) components.BreadcrumbHeaderData {
	data := pageBreadcrumbHeaderData(ctx, space, topic, editBreadcrumbMaxWidthClass, true)
	data.Items = append(data.Items, components.BreadcrumbItem{
		Label:     i18n.T(ctx, "page_edit_breadcrumb"),
		IsCurrent: true,
	})
	return data
}

// pageBreadcrumbHeaderDataはページのトピックまでのパンくずを組み立てます。ページ系画面で
// 共有します。Showはヘッダーをここから直接供給し、EditとUpdate (バリデーションエラー時に編集
// 画面を再描画する) は、現在地の項目を足すeditBreadcrumbHeaderDataを経由します。maxWidthClassは
// 各画面の本文幅にパンくずを揃えるために渡します。認証必須の編集画面はsignedInを常に真で渡し、
// 公開のShowは閲覧者の実際の状態を渡すことで、経路が認証必須画面ではなく公開スペースから
// 始まるようにします。
func pageBreadcrumbHeaderData(ctx context.Context, space viewmodel.Space, topic viewmodel.Topic, maxWidthClass string, signedIn bool) components.BreadcrumbHeaderData {
	items := append(components.HomeBreadcrumbItems(ctx, signedIn),
		components.BreadcrumbItem{
			Label: space.Name,
			Path:  templates.SpacePath(space.Identifier),
		},
		components.BreadcrumbItem{
			Label:    topic.Name,
			Path:     templates.TopicPath(space.Identifier, topic.Number),
			IconName: topic.IconName,
		},
	)

	return components.BreadcrumbHeaderData{
		MaxWidthClass: maxWidthClass,
		Items:         items,
	}
}

// editLinkResultはリンク一覧・バックリンク一覧のViewModel
type editLinkResult struct {
	LinkList     viewmodel.LinkList
	BacklinkList viewmodel.BacklinkList
}

// buildEditLinkResultInputはbuildEditLinkResultの入力。4つのページ番号は位置引数ではなく
// PageLinkStateで渡すため、呼び出し側で取り違えられない。
type buildEditLinkResultInput struct {
	LinkData        *usecase.GetEditLinkDataOutput
	SpaceIdentifier model.SpaceIdentifier
	Page            *model.Page
	State           viewmodel.PageLinkState
}

// buildEditLinkResultはUseCaseの関連ページ出力をViewModelへ変換する。
func buildEditLinkResult(input buildEditLinkResultInput) *editLinkResult {
	linkData := input.LinkData

	backlinksPerPage := make(map[model.PageID]*viewmodel.PageSliceWithCount, len(linkData.BacklinksPerPage))
	for pageID, backlinks := range linkData.BacklinksPerPage {
		backlinksPerPage[pageID] = &viewmodel.PageSliceWithCount{
			Pages:       backlinks.Pages,
			TotalCount:  backlinks.TotalCount,
			CurrentPage: input.State.BacklinkPageFor(linkData.LinkedPages, pageID),
		}
	}

	// 編集画面はページの編集権限がなければ到達しないため、一覧のカードは編集リンクを出したままに
	// する。
	editLinkData := viewmodel.BuildPageLinkData(viewmodel.BuildPageLinkDataInput{
		LinkedPages:         linkData.LinkedPages,
		LinkedTotalCount:    linkData.LinkedTotalCount,
		BacklinksPerPage:    backlinksPerPage,
		PageBacklinks:       linkData.PageBacklinks,
		PageBacklinkCount:   linkData.PageBacklinkCount,
		Topics:              linkData.LinkTopics,
		SpaceIdentifier:     input.SpaceIdentifier,
		PageNumber:          int32(input.Page.Number),
		LinkedPageFirstPage: input.State.LinkPage,
		CumulativePageLimit: input.State.CumulativePageLimit(usecase.MaxCumulativeRelatedPagePages),
		State:               input.State,
		CanEdit:             true,
	})

	return &editLinkResult{
		LinkList:     editLinkData.LinkList,
		BacklinkList: editLinkData.BacklinkList,
	}
}

// parseRelatedPageStateは3種類の関連ページ一覧について、フルページのページネーション
// フォールバックを読む。これらのクエリパラメータはhtmxで拡張した「もっと見る」リンクのネイティブな
// hrefであり、JavaScriptが使えない閲覧者も同じ範囲へ到達できる。
//
// 返す状態は正規化済みのため、UseCaseと一覧は同じページ番号で動く。
//
// 2つ目の返り値は、クエリが返せないページを指すパラメータがあったときにfalseになる。呼び出し元は
// UseCaseを呼ぶ前にこれを404へ変換する。
func parseRelatedPageState(r *http.Request, pageLinkContext viewmodel.PageLinkContext) (viewmodel.PageLinkState, bool) {
	linkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkPageQueryParam, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		return viewmodel.PageLinkState{}, false
	}

	linkedBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkedBacklinkPageQueryParam, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		return viewmodel.PageLinkState{}, false
	}

	pageBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.PageBacklinkPageQueryParam, viewmodel.RelatedPageFollowingLimit)
	if !ok {
		return viewmodel.PageLinkState{}, false
	}

	linkedPageNumber, ok := httppagination.ParseOptionalNumberParam(r, viewmodel.LinkedPageNumberQueryParam)
	if !ok {
		return viewmodel.PageLinkState{}, false
	}

	state := viewmodel.PageLinkState{
		Context:            pageLinkContext,
		LinkPage:           linkPage,
		LinkedPageNumber:   linkedPageNumber,
		LinkedBacklinkPage: linkedBacklinkPage,
		PageBacklinkPage:   pageBacklinkPage,
	}.Normalized()
	if !state.WithinCumulativeLimit(usecase.MaxCumulativeRelatedPagePages) {
		return viewmodel.PageLinkState{}, false
	}

	// フルページリクエストはリンク一覧を1ページだけ描画するため、選択カードはこのリクエストが
	// 指すページに存在する。存在しない場合はrelatedPageStateInRangeが拒否するので、状態を使う時点で
	// カードが載りうるページはこれだけになる。
	state.LinkedPageParentPage = state.LinkPage

	return state, true
}

// relatedPageCountsは範囲チェックに必要な総件数を保持する。一覧を返したUseCaseの出力から
// 取る。
type relatedPageCounts struct {
	LinkedTotalCount      int64
	PageBacklinkCount     int64
	LinkedPages           []*model.Page
	BacklinkCountByPageID map[model.PageID]int64
}

// relatedPageStateInRangeは関連ページ一覧の要求ページがすべて存在するかを返す。最終ページより
// 後ろのページは存在しない範囲を指すため、トピック詳細・スペース詳細の範囲外 ?pageと同じくページ系
// 画面も404で答える。これが無いと、古い「もっと見る」URLが一覧の消えた画面を描画してしまう。
func relatedPageStateInRange(state viewmodel.PageLinkState, counts relatedPageCounts) bool {
	if !relatedPageInRange(state.LinkPage, counts.LinkedTotalCount, viewmodel.LinkLimit) {
		return false
	}

	if !relatedPageInRange(state.PageBacklinkPage, counts.PageBacklinkCount, viewmodel.PageBacklinkLimit) {
		return false
	}

	if state.LinkedBacklinkPage <= 1 {
		return true
	}

	// 選択したカードは描画中のリンク一覧ページに載っている必要がある。載っていなければ、その
	// ネストしたページ番号は本レスポンスに含まれない一覧を指している。
	for _, linkedPage := range counts.LinkedPages {
		if int32(linkedPage.Number) != state.LinkedPageNumber {
			continue
		}
		return relatedPageInRange(state.LinkedBacklinkPage, counts.BacklinkCountByPageID[linkedPage.ID], viewmodel.BacklinkLimit)
	}

	return false
}

// relatedPageInRangeは、総件数totalCountの一覧に指定ページが存在するかを返す。initialLimitは
// 1ページあたりの件数ではなく1ページ目のカード数である。関連ページ一覧の後続ページは1件多く持つ
// ためである (viewmodel.RelatedPageTotalPagesを参照)。
func relatedPageInRange(page int32, totalCount int64, initialLimit int32) bool {
	if page <= 1 {
		return true
	}

	return int(page) <= viewmodel.RelatedPageTotalPages(totalCount, initialLimit)
}

// linkedPageBacklinkCountsは、ネストしたバックリンクの出力を範囲チェックに必要な総件数へ
// まとめる。
func linkedPageBacklinkCounts(backlinksPerPage map[model.PageID]*usecase.LinkedPageBacklinks) map[model.PageID]int64 {
	counts := make(map[model.PageID]int64, len(backlinksPerPage))
	for pageID, backlinks := range backlinksPerPage {
		counts[pageID] = backlinks.TotalCount
	}

	return counts
}
