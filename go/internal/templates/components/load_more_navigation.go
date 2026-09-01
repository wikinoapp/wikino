package components

import (
	"net/url"
	"strconv"

	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// fragmentPageQueryParam names the page of the listing a fragment endpoint returns. Each fragment
// endpoint serves exactly one listing, so its own page always travels under this name, while the
// listings it does not advance travel under the per-listing names in viewmodel (LinkPageQueryParam
// and friends), which are also the names the shared relatedPageState element gives its inputs.
//
// Keeping the two sets of names apart is what lets one editor request carry both. An editor link
// sends the shared state alongside its URL, and htmx drops a query parameter whose name the sent
// values also use, so a name in both sets would arrive holding the screen-wide value instead of the
// link's own. viewmodel.FragmentParentPageQueryParam belongs to this same fragment-scoped set.
//
// [Ja] fragmentPageQueryParam はフラグメントエンドポイントが返す一覧のページを表す。各フラグメント
// エンドポイントはちょうど 1 つの一覧を担うため、自身のページは常にこの名前で運ぶ。一方、そのリンクが
// 進めない一覧は viewmodel の一覧別の名前 (LinkPageQueryParam など) で運ぶ。これは共有の
// relatedPageState 要素が入力に付ける名前でもある。
//
// 2 組の名前を分けているからこそ、1 回の編集画面リクエストが両方を運べる。編集画面のリンクは URL と
// 一緒に共有状態を送り、htmx は送信値と同名のクエリパラメータを落とすため、両方に属する名前はリンク
// 自身の値ではなく画面全体の値として届いてしまう。viewmodel.FragmentParentPageQueryParam も、この
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

// Element ids of the editor's shared related-page state. relatedPageStateID is the container every
// editor request includes, and the three ids below are the pieces a fragment response replaces out
// of band. Each listing owns one piece, so a response never rewrites the state of a listing it did
// not advance.
//
// [Ja] 編集画面で共有する関連ページ状態の要素 ID。relatedPageStateID は編集画面の各リクエストが含める
// コンテナで、以下の 3 つはフラグメント応答が OOB で差し替える部品。1 つの一覧が 1 つの部品を持つため、
// 応答が自分の進めていない一覧の状態を書き換えることはない。
const (
	relatedPageStateID        = "page-related-page-state"
	relatedPageContextStateID = "page-related-page-context-state"
	linkPageStateID           = "page-link-list-state"
	nestedBacklinkStateID     = "page-nested-backlink-state"
	pageBacklinkStateID       = "page-backlink-list-state"
)

// relatedPageStateSelector selects the shared state element for hx-include.
//
// [Ja] relatedPageStateSelector は hx-include で共有状態の要素を選ぶ。
const relatedPageStateSelector = "#" + relatedPageStateID

// Accessible-name keys of the "load more" links. A screen renders several of them at once, so each
// listing gets its own name instead of them all reading "More" in a screen reader's link list.
//
// [Ja] 「もっと見る」リンクのアクセシブルネームのキー。1 画面に複数出るため、スクリーンリーダーの
// リンク一覧ですべてが「もっと見る」と読み上げられないよう、一覧ごとに別の名前を与える。
const (
	linkListLoadMoreAriaLabelKey         = "page_links_load_more_links_aria_label"
	backlinkListLoadMoreAriaLabelKey     = "page_links_load_more_linked_page_backlinks_aria_label"
	pageBacklinkListLoadMoreAriaLabelKey = "page_links_load_more_page_backlinks_aria_label"
)

// LoadMoreLinkData holds the fragment request and full-page fallback URLs for a related-page
// listing. htmx follows FragmentURL; the browser follows FallbackURL when JavaScript is unavailable.
//
// [Ja] LoadMoreLinkData は関連ページ一覧のフラグメント取得 URL とフルページのフォールバック URL を
// 保持する。htmx は FragmentURL を使い、JavaScript が使えないときはブラウザが FallbackURL へ遷移する。
type LoadMoreLinkData struct {
	FragmentURL string
	FallbackURL string
	Target      string
	FocusID     string
	Sync        string

	// Include selects the persistent related-page state for htmx requests and is empty when the
	// fragment URL itself carries the state.
	//
	// [Ja] Include は htmx リクエストへ含める永続的な関連ページ状態を選ぶ。フラグメント URL 自体が
	// 状態を運ぶ場合は空になる。
	Include string

	// AriaLabelKey names the listing this link advances.
	//
	// [Ja] AriaLabelKey はこのリンクが進める一覧を言い表す。
	AriaLabelKey string

	// LinkedPageTitle is the title of the listed page whose nested backlink list this link
	// advances, and is empty for the two top-level listings.
	//
	// [Ja] LinkedPageTitle は、このリンクが進めるネストしたバックリンク一覧が属するリンク先ページの
	// タイトル。最上位の 2 つの一覧では空になる。
	LinkedPageTitle string
}

// linkListLoadMoreData builds the link this listing's "load more" advances by one page.
//
// An htmx request preserves the nested and page-level backlink state because it appends cards
// without removing the old ones. How that state reaches the server differs by screen: the editor
// leaves it out of the URL and reads it from relatedPageState through hx-include, so a link already
// in the DOM still sends what another listing advanced after this link was rendered, while the
// public page has no such element and carries the state in the URL itself.
//
// The full-page fallback resets the nested state because it replaces the current link-list page and
// no longer renders the card that state names.
//
// [Ja] linkListLoadMoreData は、この一覧の「もっと見る」が 1 ページ進めるリンクを組み立てる。
//
// htmx リクエストは既存カードを残して追記するため、ネスト状態とページ自身のバックリンク状態を維持
// する。その状態がサーバーへ届く経路は画面によって異なる。編集画面は URL に載せず、hx-include で
// relatedPageState から読むため、DOM に残ったリンクでも、描画後に別の一覧が進めた値を送れる。公開
// ページにはこの要素が無いため、URL 自体で状態を運ぶ。
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

// backlinkListLoadMoreData builds the link that advances one card's nested backlink list by a page.
// The listings this link does not advance travel the same two ways as in linkListLoadMoreData: the
// editor sends them from relatedPageState through hx-include, the public page from the URL.
//
// [Ja] backlinkListLoadMoreData は、1 枚のカードのネストしたバックリンク一覧を 1 ページ進めるリンクを
// 組み立てる。このリンクが進めない一覧の状態は linkListLoadMoreData と同じ 2 通りの経路で運ぶ。編集
// 画面は hx-include で relatedPageState から送り、公開ページは URL から送る。
func backlinkListLoadMoreData(data viewmodel.BacklinkList) LoadMoreLinkData {
	nextPage := data.Pagination.Current + 1
	state := data.State
	target := backlinkListPaginationID(data.LinkedPageNumber)
	if isPaginatedEditPageLinkState(state) {
		target = backlinkListContentID(data.LinkedPageNumber)
	}

	// The card's own link-list page travels under the fragment-scoped name, because the shared state
	// the editor sends alongside carries the page of whichever card the screen currently holds open
	// (see viewmodel.FragmentParentPageQueryParam).
	//
	// [Ja] カード自身のリンク一覧ページはフラグメント側の名前で運ぶ。編集画面が一緒に送る共有状態は、
	// 画面が現在開いているカードのページを運んでいるためである
	// (viewmodel.FragmentParentPageQueryParam を参照)。
	fragmentQuery := url.Values{
		fragmentPageQueryParam:                 {strconv.Itoa(nextPage)},
		viewmodel.PageLinkContextQueryParam:    {string(viewmodel.NormalizePageLinkContext(string(state.Context)))},
		viewmodel.FragmentParentPageQueryParam: {strconv.Itoa(int(data.ParentLinkPage))},
	}
	if !isEditPageLinkState(state) {
		addLinkPageState(fragmentQuery, state)
		addPageBacklinkState(fragmentQuery, state)
	}

	// This link advances its own card's nested list, so it names that card rather than carrying the
	// nested state of whichever card the screen currently has open. The link page it carries is the
	// one holding that card, not the page the link list has since reached: a full-page request
	// renders a single link-list page, and pairing this card with a later page would name a card the
	// screen does not render.
	//
	// [Ja] このリンクは自分のカードのネストした一覧を進めるため、画面が現在開いているカードの
	// ネスト状態を引き継ぐのではなく、自分のカードを指す。運ぶリンクページも、リンク一覧が現在どこまで
	// 進んだかではなくそのカードを含むページとする。フルページリクエストはリンク一覧を 1 ページしか
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

// pageBacklinkListLoadMoreData builds the link that advances the page's own backlink list by a
// page. The listings this link does not advance travel the same two ways as in
// linkListLoadMoreData: the editor sends them from relatedPageState through hx-include, the public
// page from the URL.
//
// [Ja] pageBacklinkListLoadMoreData は、ページ自身のバックリンク一覧を 1 ページ進めるリンクを組み
// 立てる。このリンクが進めない一覧の状態は linkListLoadMoreData と同じ 2 通りの経路で運ぶ。編集画面は
// hx-include で relatedPageState から送り、公開ページは URL から送る。
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

// The limit URLs switch the editor to one-page pagination and point at the first withheld page of
// the listing that reached the cumulative safety bound. Unlike the public page, this editor context
// keeps using the member's draft link set.
//
// Each of them also carries the top-level listings it does not advance, so that switching pagination
// mode does not send the other listings back to their first page. The nested state is left out: the
// card it names belongs to a listing whose page these URLs may change, and one-page pagination
// re-renders that listing from the page they select.
//
// [Ja] 上限 URL は編集画面を 1 ページ単位のページングへ切り替え、累積取得の安全上限へ達した一覧の
// 最初に省略されたページを指す。公開ページと異なり、この編集文脈はメンバーの下書きリンク集合を使い続ける。
//
// いずれも自分が進めない最上位の一覧の状態を一緒に運び、ページング方式の切り替えで他の一覧が 1 ページ目へ
// 戻らないようにする。ネスト状態は載せない。指すカードは、これらの URL がページを変えうる一覧に属して
// おり、1 ページ単位のページングはその一覧を選択したページから描画し直すためである。
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

// The four add* helpers carry the listings a link does not advance, so that following it leaves
// them where they are. A listing sitting on its first page is left out of the query, which keeps the
// common URLs short. The selected card's parent page is the one value written even at page one, for
// the reason PageLinkState.NestedBacklinkLinkPage gives.
//
// [Ja] 4 つの add* ヘルパーは、そのリンクが進めない一覧の状態を引き継ぎ、リンクを辿っても現在位置の
// ままにする。1 ページ目の一覧はクエリに載せないため、通常の URL は短いままになる。選択カードの親ページ
// だけは 1 ページ目でも書き出す。理由は PageLinkState.NestedBacklinkLinkPage を参照。
func addLinkPageState(query url.Values, state viewmodel.PageLinkState) {
	if state.LinkPage > 1 {
		query.Set(viewmodel.LinkPageQueryParam, strconv.Itoa(int(state.LinkPage)))
	}
}

// addNestedBacklinkState carries the selected card and its nested backlink page into a fragment
// request, together with the link-list page holding that card. The parent page rides along because
// the response builds its own full-page fallback, which has to render the page the card is on.
//
// [Ja] addNestedBacklinkState は、選択したカードとそのネストしたバックリンクのページを、カードを含む
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

// addLinkAndNestedBacklinkState carries the link list's page and, when one is selected, the card
// whose nested backlink list has been advanced. The two travel together in a full-page URL: a card
// exists only on the link-list page it was rendered from, so the selected card's page wins over
// whichever page the link list has since reached. The link page a full-page request renders is what
// decides which cards it holds, so the parent page itself needs no separate parameter here.
//
// [Ja] addLinkAndNestedBacklinkState はリンク一覧のページと、選択されていればネストしたバックリンク
// 一覧を進めたカードを一緒に運ぶ。フルページ URL では両者が組で意味を持つ。カードは描画元のページに
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
