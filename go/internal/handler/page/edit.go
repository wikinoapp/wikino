package page

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/httppagination"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// zenModeCookieName is the cookie holding the editor's Zen mode state ("1" = on; off is the
// cookie's absence). It is written by web/zen-mode.ts, so it is not HttpOnly; keep the name in
// sync with that script.
//
// [Ja] zenModeCookieName はエディタの Zenモード状態を保持するクッキー ("1" で ON。OFF は
// クッキーなし)。web/zen-mode.ts が書き込むため HttpOnly ではない。名前は同スクリプトと
// 同期させること。
const zenModeCookieName = "wikino_zen_mode"

// zenModeFromRequest reads the Zen mode state from the request cookie.
// [Ja] zenModeFromRequest はリクエストのクッキーから Zenモード状態を読み取ります。
func zenModeFromRequest(r *http.Request) bool {
	cookie, err := r.Cookie(zenModeCookieName)
	return err == nil && cookie.Value == "1"
}

// Edit はページ編集フォームを表示します (GET /s/{space_identifier}/pages/{page_number}/edit)
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

	// Build links from the stored identifier, not from the URL, so that every link on the screen uses
	// the same form.
	//
	// [Ja] URL ではなく保存済みの識別子からリンクを組み立て、画面内のリンクの表記を揃える。
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
		// The editor stays within a single space, so omit the space label on each draft card.
		// [Ja] 編集画面は同一スペース内のため、各下書きカードのスペースラベルを省く。
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

	// The editor supplies the global-nav state via GlobalNav. PageNamePageEdit matches no nav item,
	// so no item is highlighted (the draft list column, not the nav, handles in-screen navigation).
	//
	// [Ja] 編集画面はグローバルナビの状態を GlobalNav で供給する。PageNamePageEdit はどのナビ項目にも
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

		BreadcrumbHeader: pageBreadcrumbHeaderData(ctx, spaceVM, topicVM, editBreadcrumbMaxWidthClass, true),
	}

	err = layouts.Default(layoutData, content).Render(ctx, w)
	if err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// editBreadcrumbMaxWidthClass lines the editor's breadcrumb up with its wider body.
//
// [Ja] editBreadcrumbMaxWidthClass は編集画面の広い本文幅にパンくずを揃える。
const editBreadcrumbMaxWidthClass = "max-w-6xl"

// pageBreadcrumbHeaderData builds the breadcrumb header the layout renders for the page screens.
// Edit, Update (which re-renders the editor on a validation error) and Show all supply the header
// from here; maxWidthClass lines the breadcrumb up with each screen's body. The authenticated Edit
// and Update pass signedIn unconditionally, while the public Show passes the viewer's actual state
// so the trail starts at the public space rather than at an authenticated-only screen.
//
// [Ja] pageBreadcrumbHeaderData はページ系画面のパンくずヘッダーを組み立てます。描画するのは
// レイアウトです。Edit・Update (バリデーションエラー時に編集画面を再描画する)・Show のいずれも
// ヘッダーをここから供給します。maxWidthClass は各画面の本文幅にパンくずを揃えるために渡します。
// 認証必須の Edit・Update は signedIn を常に真で渡し、公開の Show は閲覧者の実際の状態を渡すことで、
// 経路が認証必須画面ではなく公開スペースから始まるようにします。
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

// editLinkResult はリンク一覧・バックリンク一覧のViewModel
type editLinkResult struct {
	LinkList     viewmodel.LinkList
	BacklinkList viewmodel.BacklinkList
}

// buildEditLinkResultInput holds the input of buildEditLinkResult. The four page numbers travel in
// PageLinkState rather than as positional arguments, so a call site cannot mix them up.
//
// [Ja] buildEditLinkResultInput は buildEditLinkResult の入力。4 つのページ番号は位置引数ではなく
// PageLinkState で渡すため、呼び出し側で取り違えられない。
type buildEditLinkResultInput struct {
	LinkData        *usecase.GetEditLinkDataOutput
	SpaceIdentifier model.SpaceIdentifier
	Page            *model.Page
	State           viewmodel.PageLinkState
}

// buildEditLinkResult converts the usecase's related-page output into view models.
//
// [Ja] buildEditLinkResult は UseCase の関連ページ出力を ViewModel へ変換する。
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

	// The editor is only reachable with edit rights on the page, so the listed cards keep their
	// edit links.
	//
	// [Ja] 編集画面はページの編集権限がなければ到達しないため、一覧のカードは編集リンクを出したままに
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

// parseRelatedPageState reads the full-page pagination fallback of the three related-page listings.
// These query parameters are the native href behind the htmx-enhanced "load more" links, so a
// viewer without JavaScript reaches the same slices through them.
//
// The state comes back normalized, so the usecase and the listings work from the same page numbers.
//
// The second return value is false when a parameter names a page the queries cannot serve, which
// the caller turns into a 404 before invoking the usecase.
//
// [Ja] parseRelatedPageState は 3 種類の関連ページ一覧について、フルページのページネーション
// フォールバックを読む。これらのクエリパラメータは htmx で拡張した「もっと見る」リンクのネイティブな
// href であり、JavaScript が使えない閲覧者も同じ範囲へ到達できる。
//
// 返す状態は正規化済みのため、UseCase と一覧は同じページ番号で動く。
//
// 2 つ目の返り値は、クエリが返せないページを指すパラメータがあったときに false になる。呼び出し元は
// UseCase を呼ぶ前にこれを 404 へ変換する。
func parseRelatedPageState(r *http.Request, pageLinkContext viewmodel.PageLinkContext) (viewmodel.PageLinkState, bool) {
	linkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkPageQueryParam, viewmodel.LinkLimit)
	if !ok {
		return viewmodel.PageLinkState{}, false
	}

	linkedBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.LinkedBacklinkPageQueryParam, viewmodel.BacklinkLimit)
	if !ok {
		return viewmodel.PageLinkState{}, false
	}

	pageBacklinkPage, ok := httppagination.ParseNamedPageParam(r, viewmodel.PageBacklinkPageQueryParam, viewmodel.PageBacklinkLimit)
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

	// A full-page request renders exactly one link-list page, so the selected card lives on the page
	// this request names. relatedPageStateInRange rejects the request when it does not, which makes
	// this the only page the card can be on by the time the state is used.
	//
	// [Ja] フルページリクエストはリンク一覧を 1 ページだけ描画するため、選択カードはこのリクエストが
	// 指すページに存在する。存在しない場合は relatedPageStateInRange が拒否するので、状態を使う時点で
	// カードが載りうるページはこれだけになる。
	state.LinkedPageParentPage = state.LinkPage

	return state, true
}

// relatedPageCounts holds the totals the range check needs, taken from whichever usecase produced
// the listings.
//
// [Ja] relatedPageCounts は範囲チェックに必要な総件数を保持する。一覧を返した UseCase の出力から
// 取る。
type relatedPageCounts struct {
	LinkedTotalCount      int64
	PageBacklinkCount     int64
	LinkedPages           []*model.Page
	BacklinkCountByPageID map[model.PageID]int64
}

// relatedPageStateInRange reports whether every requested page of the related-page listings exists.
// A page past the last one names a slice that is not there, which the page screens answer with a
// 404 the same way the topic and space screens answer an out-of-range ?page. Without this a stale
// "load more" URL would render the screen with its listing silently missing.
//
// [Ja] relatedPageStateInRange は関連ページ一覧の要求ページがすべて存在するかを返す。最終ページより
// 後ろのページは存在しない範囲を指すため、トピック詳細・スペース詳細の範囲外 ?page と同じくページ系
// 画面も 404 で答える。これが無いと、古い「もっと見る」URL が一覧の消えた画面を描画してしまう。
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

	// The selected card must be on the link-list page being rendered; otherwise its nested page
	// number names a listing this response does not contain.
	//
	// [Ja] 選択したカードは描画中のリンク一覧ページに載っている必要がある。載っていなければ、その
	// ネストしたページ番号は本レスポンスに含まれない一覧を指している。
	for _, linkedPage := range counts.LinkedPages {
		if int32(linkedPage.Number) != state.LinkedPageNumber {
			continue
		}
		return relatedPageInRange(state.LinkedBacklinkPage, counts.BacklinkCountByPageID[linkedPage.ID], viewmodel.BacklinkLimit)
	}

	return false
}

// relatedPageInRange reports whether the given page of a listing of totalCount items exists.
//
// [Ja] relatedPageInRange は、総件数 totalCount の一覧に指定ページが存在するかを返す。
func relatedPageInRange(page int32, totalCount int64, limit int32) bool {
	if page <= 1 {
		return true
	}

	return int(page) <= viewmodel.NewPagination(1, totalCount, int(limit)).Total
}

// linkedPageBacklinkCounts reduces the nested backlink output to the totals the range check needs.
//
// [Ja] linkedPageBacklinkCounts は、ネストしたバックリンクの出力を範囲チェックに必要な総件数へ
// まとめる。
func linkedPageBacklinkCounts(backlinksPerPage map[model.PageID]*usecase.LinkedPageBacklinks) map[model.PageID]int64 {
	counts := make(map[model.PageID]int64, len(backlinksPerPage))
	for pageID, backlinks := range backlinksPerPage {
		counts[pageID] = backlinks.TotalCount
	}

	return counts
}
