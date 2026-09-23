package components

import (
	"net/url"
	"strconv"

	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// fragmentPageQueryParamはフラグメントエンドポイントが返す一覧のページを表す。各フラグメント
// エンドポイントはちょうど1つの一覧を担うため、自身のページは常にこの名前で運ぶ。一方、そのリンクが
// 進めない一覧はviewmodelの一覧別の名前 (LinkPageQueryParamなど) で運ぶ。これは共有の
// relatedPageState要素が入力に付ける名前でもある。
//
// 2組の名前を分けているからこそ、1回の編集画面リクエストが両方を運べる。編集画面のリンクはURLと
// 一緒に共有状態を送り、htmxは送信値と同名のクエリパラメータを落とすため、両方に属する名前はリンク
// 自身の値ではなく画面全体の値として届いてしまう。viewmodel.FragmentParentPageQueryParamも、この
// フラグメント側の組に属する。
const fragmentPageQueryParam = "page"

const (
	linkListContainerID            = "page-link-list"
	linkListPaginationID           = "page-link-list-pagination"
	linkListFallbackAnchorID       = "page-link-list-content"
	linkListLoadMoreFocusID        = "page-link-list-load-more"
	relatedLinkListID              = "page-related-link-list"
	pageBacklinkListContainerID    = "page-backlink-list"
	pageBacklinkPaginationID       = "page-backlink-list-pagination"
	pageBacklinkFallbackAnchorID   = "page-backlink-list-content"
	pageBacklinkLoadMoreFocusID    = "page-backlink-list-load-more"
	draftPageRefreshTriggerID      = "page-draft-refresh-trigger"
	loadMoreRelatedPageRequestSync = "#page-related-page-state:replace"
	draftRelatedPageRequestSync    = "#page-related-page-state:drop"
)

// 編集画面で共有する関連ページ状態の要素ID。relatedPageStateIDは編集画面の各リクエストが含める
// コンテナで、以下の3つはフラグメント応答がOOBで差し替える部品。1つの一覧が1つの部品を持つため、
// 応答が自分の進めていない一覧の状態を書き換えることはない。
const (
	relatedPageStateID        = "page-related-page-state"
	relatedPageContextStateID = "page-related-page-context-state"
	linkPageStateID           = "page-link-list-state"
	nestedBacklinkStateID     = "page-nested-backlink-state"
	pageBacklinkStateID       = "page-backlink-list-state"
)

// relatedPageStateSelectorはhx-includeで共有状態の要素を選ぶ。
const relatedPageStateSelector = "#" + relatedPageStateID

// 「もっと見る」リンクのアクセシブルネームのキー。1画面に複数出るため、スクリーンリーダーの
// リンク一覧ですべてが「もっと見る」と読み上げられないよう、一覧ごとに別の名前を与える。
const (
	linkListLoadMoreAriaLabelKey         = "page_links_load_more_links_aria_label"
	backlinkListLoadMoreAriaLabelKey     = "page_links_load_more_linked_page_backlinks_aria_label"
	pageBacklinkListLoadMoreAriaLabelKey = "page_links_load_more_page_backlinks_aria_label"
)

// LoadMoreLinkDataは関連ページ一覧のフラグメント取得URLとフルページのフォールバックURLを
// 保持する。htmxはFragmentURLを使い、JavaScriptが使えないときはブラウザがFallbackURLへ遷移する。
type LoadMoreLinkData struct {
	FragmentURL string
	FallbackURL string
	Target      string
	FocusID     string
	Sync        string

	// Includeはhtmxリクエストへ含める永続的な関連ページ状態を選ぶ。フラグメントURL自体が
	// 状態を運ぶ場合は空になる。
	Include string

	// AriaLabelKeyはこのリンクが進める一覧を言い表す。
	AriaLabelKey string

	// LinkedPageTitleは、このリンクが進めるネストしたバックリンク一覧が属するリンク先ページの
	// タイトル。最上位の2つの一覧では空になる。
	LinkedPageTitle string
}

// linkListLoadMoreDataは、この一覧の「もっと見る」が1ページ進めるリンクを組み立てる。
//
// htmxリクエストは既存カードを残して追記するため、ネスト状態とページ自身のバックリンク状態を維持
// する。その状態がサーバーへ届く経路は画面によって異なる。編集画面はURLに載せず、hx-includeで
// relatedPageStateから読むため、DOMに残ったリンクでも、描画後に別の一覧が進めた値を送れる。公開
// ページにはこの要素が無いため、URL自体で状態を運ぶ。
//
// フルページフォールバックは現在のリンク一覧ページを入れ替え、状態が指すカードを描画しなくなるため、
// ネスト状態だけをリセットする。
func linkListLoadMoreData(data viewmodel.LinkList) LoadMoreLinkData {
	nextPage := data.Pagination.Current + 1
	state := data.State
	target := linkListPaginationID
	if isPaginatedEditPageLinkState(state) {
		target = linkListFallbackAnchorID
	}

	fragmentQuery := url.Values{
		fragmentPageQueryParam:              {strconv.Itoa(nextPage)},
		viewmodel.PageLinkContextQueryParam: {string(viewmodel.NormalizePageLinkContext(string(state.Context)))},
	}
	if !isEditPageLinkState(state) {
		addNestedBacklinkState(fragmentQuery, state)
		addPageBacklinkState(fragmentQuery, state)
	}

	fallbackQuery := url.Values{
		viewmodel.LinkPageQueryParam: {strconv.Itoa(nextPage)},
	}
	addPageBacklinkState(fallbackQuery, state)
	addPaginatedEditContext(fallbackQuery, state)

	return LoadMoreLinkData{
		FragmentURL:  relatedPageURL(string(templates.PageLinkListPath(data.SpaceIdentifier, data.PageNumber)), fragmentQuery, ""),
		FallbackURL:  relatedPageURL(parentPagePath(state, data.SpaceIdentifier, data.PageNumber), fallbackQuery, linkListFallbackAnchorID),
		Target:       "#" + target,
		FocusID:      linkListLoadMoreFocusID,
		Include:      editRelatedPageStateInclude(state),
		Sync:         editRelatedPageSync(state),
		AriaLabelKey: linkListLoadMoreAriaLabelKey,
	}
}

// backlinkListLoadMoreDataは、1枚のカードのネストしたバックリンク一覧を1ページ進めるリンクを
// 組み立てる。このリンクが進めない一覧の状態はlinkListLoadMoreDataと同じ2通りの経路で運ぶ。編集
// 画面はhx-includeでrelatedPageStateから送り、公開ページはURLから送る。
func backlinkListLoadMoreData(data viewmodel.BacklinkList) LoadMoreLinkData {
	nextPage := data.Pagination.Current + 1
	state := data.State
	target := backlinkListPaginationID(data.LinkedPageNumber)
	if isPaginatedEditPageLinkState(state) {
		target = backlinkListContentID(data.LinkedPageNumber)
	}

	// カード自身のリンク一覧ページはフラグメント側の名前で運ぶ。編集画面が一緒に送る共有状態は、
	// 画面が現在開いているカードのページを運んでいるためである
	// (viewmodel.FragmentParentPageQueryParamを参照)。
	fragmentQuery := url.Values{
		fragmentPageQueryParam:                 {strconv.Itoa(nextPage)},
		viewmodel.PageLinkContextQueryParam:    {string(viewmodel.NormalizePageLinkContext(string(state.Context)))},
		viewmodel.FragmentParentPageQueryParam: {strconv.Itoa(int(data.ParentLinkPage))},
	}
	if !isEditPageLinkState(state) {
		addLinkPageState(fragmentQuery, state)
		addPageBacklinkState(fragmentQuery, state)
	}

	// このリンクは自分のカードのネストした一覧を進めるため、画面が現在開いているカードの
	// ネスト状態を引き継ぐのではなく、自分のカードを指す。運ぶリンクページも、リンク一覧が現在どこまで
	// 進んだかではなくそのカードを含むページとする。フルページリクエストはリンク一覧を1ページしか
	// 描画せず、後続ページと組み合わせると描画対象にないカードを指してしまうためである。
	fallbackQuery := url.Values{
		viewmodel.LinkedPageNumberQueryParam:   {strconv.Itoa(int(data.LinkedPageNumber))},
		viewmodel.LinkedBacklinkPageQueryParam: {strconv.Itoa(nextPage)},
	}
	if data.ParentLinkPage > 1 {
		fallbackQuery.Set(viewmodel.LinkPageQueryParam, strconv.Itoa(int(data.ParentLinkPage)))
	}
	addPageBacklinkState(fallbackQuery, state)
	addPaginatedEditContext(fallbackQuery, state)

	return LoadMoreLinkData{
		FragmentURL: relatedPageURL(
			string(templates.PageBacklinkListPath(data.SpaceIdentifier, data.PageNumber, data.LinkedPageNumber)),
			fragmentQuery,
			"",
		),
		FallbackURL:     relatedPageURL(parentPagePath(state, data.SpaceIdentifier, data.PageNumber), fallbackQuery, linkListItemFallbackAnchorID(data.LinkedPageNumber)),
		Target:          "#" + target,
		FocusID:         backlinkListLoadMoreFocusID(data.LinkedPageNumber),
		Include:         editRelatedPageStateInclude(state),
		Sync:            editRelatedPageSync(state),
		AriaLabelKey:    backlinkListLoadMoreAriaLabelKey,
		LinkedPageTitle: data.LinkedPageTitle,
	}
}

// pageBacklinkListLoadMoreDataは、ページ自身のバックリンク一覧を1ページ進めるリンクを組み
// 立てる。このリンクが進めない一覧の状態はlinkListLoadMoreDataと同じ2通りの経路で運ぶ。編集画面は
// hx-includeでrelatedPageStateから送り、公開ページはURLから送る。
func pageBacklinkListLoadMoreData(data viewmodel.BacklinkList) LoadMoreLinkData {
	nextPage := data.Pagination.Current + 1
	state := data.State
	target := pageBacklinkPaginationID
	if isPaginatedEditPageLinkState(state) {
		target = pageBacklinkFallbackAnchorID
	}

	fragmentQuery := url.Values{
		fragmentPageQueryParam:              {strconv.Itoa(nextPage)},
		viewmodel.PageLinkContextQueryParam: {string(viewmodel.NormalizePageLinkContext(string(state.Context)))},
	}
	if !isEditPageLinkState(state) {
		addLinkPageState(fragmentQuery, state)
		addNestedBacklinkState(fragmentQuery, state)
	}

	fallbackQuery := url.Values{
		viewmodel.PageBacklinkPageQueryParam: {strconv.Itoa(nextPage)},
	}
	addLinkAndNestedBacklinkState(fallbackQuery, state)
	addPaginatedEditContext(fallbackQuery, state)

	return LoadMoreLinkData{
		FragmentURL:  relatedPageURL(string(templates.PageBacklinksPath(data.SpaceIdentifier, data.PageNumber)), fragmentQuery, ""),
		FallbackURL:  relatedPageURL(parentPagePath(state, data.SpaceIdentifier, data.PageNumber), fallbackQuery, pageBacklinkFallbackAnchorID),
		Target:       "#" + target,
		FocusID:      pageBacklinkLoadMoreFocusID,
		Include:      editRelatedPageStateInclude(state),
		Sync:         editRelatedPageSync(state),
		AriaLabelKey: pageBacklinkListLoadMoreAriaLabelKey,
	}
}

func isEditPageLinkState(state viewmodel.PageLinkState) bool {
	return viewmodel.NormalizePageLinkContext(string(state.Context)).IsEdit()
}

func isPaginatedEditPageLinkState(state viewmodel.PageLinkState) bool {
	return viewmodel.NormalizePageLinkContext(string(state.Context)) == viewmodel.PageLinkContextEditPaginated
}

func editRelatedPageStateInclude(state viewmodel.PageLinkState) string {
	if !isEditPageLinkState(state) {
		return ""
	}

	return relatedPageStateSelector
}

func editRelatedPageSync(state viewmodel.PageLinkState) string {
	if !isEditPageLinkState(state) {
		return ""
	}

	return loadMoreRelatedPageRequestSync
}

// 上限URLは編集画面を1ページ単位のページングへ切り替え、累積取得の安全上限へ達した一覧の
// 最初に省略されたページを指す。公開ページと異なり、この編集文脈はメンバーの下書きリンク集合を使い続ける。
//
// いずれも自分が進めない最上位の一覧の状態を一緒に運び、ページング方式の切り替えで他の一覧が1ページ目へ
// 戻らないようにする。ネスト状態は載せない。指すカードは、これらのURLがページを変えうる一覧に属して
// おり、1ページ単位のページングはその一覧を選択したページから描画し直すためである。
func linkListLimitURL(data viewmodel.LinkList) string {
	query := paginatedEditQuery()
	query.Set(viewmodel.LinkPageQueryParam, strconv.Itoa(data.Pagination.Current+1))
	addPageBacklinkState(query, data.State)
	return relatedPageURL(string(templates.PageEditPath(data.SpaceIdentifier, data.PageNumber)), query, linkListFallbackAnchorID)
}

func backlinkListLimitURL(data viewmodel.BacklinkList) string {
	query := paginatedEditQuery()
	query.Set(viewmodel.LinkPageQueryParam, strconv.Itoa(int(data.ParentLinkPage)))
	query.Set(viewmodel.LinkedPageNumberQueryParam, strconv.Itoa(int(data.LinkedPageNumber)))
	query.Set(viewmodel.LinkedBacklinkPageQueryParam, strconv.Itoa(data.Pagination.Current+1))
	addPageBacklinkState(query, data.State)
	return relatedPageURL(string(templates.PageEditPath(data.SpaceIdentifier, data.PageNumber)), query, linkListItemFallbackAnchorID(data.LinkedPageNumber))
}

func pageBacklinkListLimitURL(data viewmodel.BacklinkList) string {
	query := paginatedEditQuery()
	query.Set(viewmodel.PageBacklinkPageQueryParam, strconv.Itoa(data.Pagination.Current+1))
	addLinkPageState(query, data.State)
	return relatedPageURL(string(templates.PageEditPath(data.SpaceIdentifier, data.PageNumber)), query, pageBacklinkFallbackAnchorID)
}

func paginatedEditQuery() url.Values {
	return url.Values{
		viewmodel.PageLinkContextQueryParam: {string(viewmodel.PageLinkContextEditPaginated)},
	}
}

func addPaginatedEditContext(query url.Values, state viewmodel.PageLinkState) {
	if isPaginatedEditPageLinkState(state) {
		query.Set(viewmodel.PageLinkContextQueryParam, string(viewmodel.PageLinkContextEditPaginated))
	}
}

func backlinkListPaginationID(linkedPageNumber int32) string {
	return "page-backlink-list-" + strconv.Itoa(int(linkedPageNumber))
}

func backlinkListContentID(linkedPageNumber int32) string {
	return backlinkListPaginationID(linkedPageNumber) + "-content"
}

func backlinkListLoadMoreFocusID(linkedPageNumber int32) string {
	return backlinkListPaginationID(linkedPageNumber) + "-load-more"
}

func linkListItemFallbackAnchorID(linkedPageNumber int32) string {
	return "page-link-list-item-" + strconv.Itoa(int(linkedPageNumber))
}

// 4つのadd* ヘルパーは、そのリンクが進めない一覧の状態を引き継ぎ、リンクを辿っても現在位置の
// ままにする。1ページ目の一覧はクエリに載せないため、通常のURLは短いままになる。選択カードの親ページ
// だけは1ページ目でも書き出す。理由はPageLinkState.NestedBacklinkLinkPageを参照。
func addLinkPageState(query url.Values, state viewmodel.PageLinkState) {
	if state.LinkPage > 1 {
		query.Set(viewmodel.LinkPageQueryParam, strconv.Itoa(int(state.LinkPage)))
	}
}

// addNestedBacklinkStateは、選択したカードとそのネストしたバックリンクのページを、カードを含む
// リンク一覧のページと一緒にフラグメントリクエストへ載せる。親ページを一緒に運ぶのは、応答が自身の
// フルページフォールバックを組み立てる際に、カードの載るページを描画する必要があるためである。
func addNestedBacklinkState(query url.Values, state viewmodel.PageLinkState) {
	if !state.HasNestedBacklinkState() {
		return
	}

	query.Set(viewmodel.LinkedPageNumberQueryParam, strconv.Itoa(int(state.LinkedPageNumber)))
	query.Set(viewmodel.LinkedBacklinkPageQueryParam, strconv.Itoa(int(state.LinkedBacklinkPage)))
	query.Set(viewmodel.LinkedPageParentPageQueryParam, strconv.Itoa(int(state.NestedBacklinkLinkPage())))
}

// addLinkAndNestedBacklinkStateはリンク一覧のページと、選択されていればネストしたバックリンク
// 一覧を進めたカードを一緒に運ぶ。フルページURLでは両者が組で意味を持つ。カードは描画元のページに
// しか存在しないため、リンク一覧が現在どこまで進んだかより選択カードのページを優先する。フルページ
// リクエストが描画するリンクページがどのカードを含むかを決めるため、親ページ自体は別のパラメータに
// しなくてよい。
func addLinkAndNestedBacklinkState(query url.Values, state viewmodel.PageLinkState) {
	if !state.HasNestedBacklinkState() {
		addLinkPageState(query, state)
		return
	}

	query.Set(viewmodel.LinkedPageNumberQueryParam, strconv.Itoa(int(state.LinkedPageNumber)))
	query.Set(viewmodel.LinkedBacklinkPageQueryParam, strconv.Itoa(int(state.LinkedBacklinkPage)))

	if linkPage := state.NestedBacklinkLinkPage(); linkPage > 1 {
		query.Set(viewmodel.LinkPageQueryParam, strconv.Itoa(int(linkPage)))
	}
}

func addPageBacklinkState(query url.Values, state viewmodel.PageLinkState) {
	if state.PageBacklinkPage > 1 {
		query.Set(viewmodel.PageBacklinkPageQueryParam, strconv.Itoa(int(state.PageBacklinkPage)))
	}
}

func parentPagePath(state viewmodel.PageLinkState, spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) string {
	if viewmodel.NormalizePageLinkContext(string(state.Context)) == viewmodel.PageLinkContextShow {
		return string(templates.PagePath(spaceIdentifier, viewmodel.PageNumber(pageNumber)))
	}
	return string(templates.PageEditPath(spaceIdentifier, pageNumber))
}

func relatedPageURL(path string, query url.Values, fragment string) string {
	result := path + "?" + query.Encode()
	if fragment != "" {
		result += "#" + fragment
	}
	return result
}
