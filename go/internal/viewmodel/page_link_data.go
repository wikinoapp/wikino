package viewmodel

import (
	"math"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// PageSliceWithCount pairs one related-page slice with its total count and current page.
//
// [Ja] PageSliceWithCount は関連ページ一覧の 1 範囲と総件数・現在ページを組み合わせる。
type PageSliceWithCount struct {
	Pages       []*model.Page
	TotalCount  int64
	CurrentPage int32
}

// PageLinkContext identifies the full page that owns a related-page listing. Fragment URLs carry
// this value so their progressively enhanced links can fall back to the correct parent screen.
//
// [Ja] PageLinkContext は関連ページ一覧を持つフルページを表す。フラグメント URL がこの値を
// 引き継ぐことで、プログレッシブエンハンスメントされたリンクは正しい親画面へフォールバックできる。
type PageLinkContext string

const (
	// PageLinkContextEdit is the page editor, whose link list may follow the member's draft.
	//
	// [Ja] PageLinkContextEdit はメンバーの下書きをリンク一覧へ反映するページ編集画面。
	PageLinkContextEdit PageLinkContext = "edit"

	// PageLinkContextEditPaginated is the page editor after a cumulative listing reaches its safety
	// limit. Every related-page listing then renders one requested page at a time, so the member can
	// keep following draft-only links without making a draft refresh fetch an unbounded prefix.
	//
	// [Ja] PageLinkContextEditPaginated は、累積一覧が安全上限へ達したあとのページ編集画面。各関連
	// ページ一覧を要求された 1 ページずつ描画し、下書き再取得に無制限な先頭範囲を取得させずに、
	// メンバーが下書き固有のリンクを辿り続けられるようにする。
	PageLinkContextEditPaginated PageLinkContext = "edit_paginated"

	// PageLinkContextShow is the public page detail, whose link list always uses saved links.
	//
	// [Ja] PageLinkContextShow は保存済みリンクだけを使う公開ページ表示画面。
	PageLinkContextShow PageLinkContext = "show"
)

// Query parameter names shared by the related-page listings. The screens that render the listings
// and the fragment endpoints that continue them both refer to these, so a rename cannot leave the
// two sides disagreeing.
//
// [Ja] 関連ページ一覧で共有するクエリパラメータ名。一覧を描画する画面と、その続きを返す
// フラグメントエンドポイントの双方がこれを参照するため、リネームしても両者がずれない。
const (
	// PageLinkContextQueryParam carries PageLinkContext from a screen to its fragments.
	//
	// [Ja] PageLinkContextQueryParam は画面からフラグメントへ PageLinkContext を伝える。
	PageLinkContextQueryParam = "context"

	// LinkPageQueryParam selects the page of the link list on a full-page request.
	//
	// [Ja] LinkPageQueryParam はフルページリクエストでリンク一覧のページを選ぶ。
	LinkPageQueryParam = "links_page"

	// LinkedPageNumberQueryParam selects which listed page's nested backlink list is advanced.
	//
	// [Ja] LinkedPageNumberQueryParam は、どのリンク先ページのネストしたバックリンク一覧を
	// 進めるかを選ぶ。
	LinkedPageNumberQueryParam = "linked_page_number"

	// LinkedBacklinkPageQueryParam selects the page of that nested backlink list.
	//
	// [Ja] LinkedBacklinkPageQueryParam はそのネストしたバックリンク一覧のページを選ぶ。
	LinkedBacklinkPageQueryParam = "linked_backlinks_page"

	// LinkedPageParentPageQueryParam carries the link-list page containing the selected card. The
	// value belongs to that card's nested listing, but editor requests carry it in the nested portion
	// of the screen-wide shared state.
	//
	// [Ja] LinkedPageParentPageQueryParam は選択したカードを含むリンク一覧のページ番号を運ぶ。この値は
	// そのカードのネストした一覧に属するが、編集画面のリクエストでは画面全体の共有状態のうち
	// ネスト部分として運ぶ。
	LinkedPageParentPageQueryParam = "linked_page_parent_page"

	// FragmentParentPageQueryParam carries the same kind of value as LinkedPageParentPageQueryParam,
	// but names the card a nested backlink fragment advances rather than the card the screen
	// currently holds open. The two need separate names: an editor request also sends the
	// screen-wide state, and htmx replaces a query parameter with the included value of the same
	// name, so sharing one name would drop the page belonging to the card being advanced. It is the
	// same split that keeps a fragment's own "page" apart from the per-listing page names.
	//
	// [Ja] FragmentParentPageQueryParam は LinkedPageParentPageQueryParam と同種の値を運ぶが、指すのは
	// 画面が現在開いているカードではなく、ネストしたバックリンクのフラグメントが進めるカードである。
	// 名前を分ける必要があるのは、編集画面のリクエストが画面全体の状態も送っており、htmx が同名の
	// 送信値でクエリパラメータを置き換えるためである。名前を共有すると、進めるカード自身のページが
	// 失われる。フラグメント自身の "page" を一覧ごとのページ名と分けているのと同じ切り分けである。
	FragmentParentPageQueryParam = "parent_page"

	// PageBacklinkPageQueryParam selects the page of the page's own backlink list.
	//
	// [Ja] PageBacklinkPageQueryParam はページ自身のバックリンク一覧のページを選ぶ。
	PageBacklinkPageQueryParam = "backlinks_page"
)

// NormalizePageLinkContext resolves a raw query value to a context. Anything other than the page
// detail screen's value means the editor, so a missing or tampered parameter falls back to the
// screen whose listing may follow a draft rather than silently widening what a fragment returns.
//
// [Ja] NormalizePageLinkContext は生のクエリ値を文脈へ解決する。ページ表示画面の値以外はすべて
// 編集画面として扱うため、パラメータが無い場合や壊れている場合は、フラグメントの返す範囲を
// 黙って広げるのではなく下書きを反映しうる画面へフォールバックする。
func NormalizePageLinkContext(value string) PageLinkContext {
	switch PageLinkContext(value) {
	case PageLinkContextShow:
		return PageLinkContextShow
	case PageLinkContextEditPaginated:
		return PageLinkContextEditPaginated
	default:
		return PageLinkContextEdit
	}
}

// IsEdit reports whether the context belongs to either editor pagination mode.
//
// [Ja] IsEdit は、編集画面のどちらかのページング方式に属する文脈かを返す。
func (c PageLinkContext) IsEdit() bool {
	return NormalizePageLinkContext(string(c)) != PageLinkContextShow
}

// IncludesPrecedingPages reports whether a draft refresh must rebuild the whole loaded prefix. The
// paginated editor and the public page replace one requested page instead.
//
// [Ja] IncludesPrecedingPages は、下書き再取得が読み込み済みの先頭範囲全体を再構築する必要があるかを
// 返す。ページ単位の編集画面と公開ページは、要求された 1 ページだけを差し替える。
func (c PageLinkContext) IncludesPrecedingPages() bool {
	return NormalizePageLinkContext(string(c)) == PageLinkContextEdit
}

// PageLinkState is the pagination state of every related-page listing on one screen. Each "load
// more" link advances a single listing and carries independent listings unchanged. Appending the
// parent link list with htmx preserves the nested backlink state because existing cards remain;
// following a full-page fallback that replaces the parent page resets it because that child belongs
// to a card on the page being replaced.
//
// Only one listed page's nested backlink list can be advanced at a time, because the state names it
// with a single page number.
//
// [Ja] PageLinkState は 1 画面上の関連ページ一覧すべてのページネーション状態。各「もっと見る」
// リンクは 1 つの一覧だけを進め、独立した一覧の状態はそのまま引き継ぐ。htmx で親のリンク一覧を
// 追記するときは既存のカードが残るためネストしたバックリンク状態を維持し、親ページを置き換える
// フルページフォールバックでは、入れ替わるカードに従属する子状態をリセットする。
//
// ネストしたバックリンク一覧を進められるのは一度に 1 つのリンク先ページだけである。状態が
// それをページ番号 1 つで指しているためである。
type PageLinkState struct {
	Context            PageLinkContext
	LinkPage           int32
	LinkedPageNumber   int32
	LinkedBacklinkPage int32
	PageBacklinkPage   int32

	// LinkedPageParentPage is the link-list page holding the card named by LinkedPageNumber, and is
	// zero while no card is selected. A card exists only on the page it was rendered from, so a
	// full-page URL that carries the nested state has to render that page rather than whichever page
	// the link list has since reached; the screens answer a nested page number naming an absent card
	// with a 404.
	//
	// [Ja] LinkedPageParentPage は LinkedPageNumber が指すカードを含むリンク一覧のページ。カードが
	// 選ばれていないときはゼロになる。カードは描画元のページにしか存在しないため、ネスト状態を運ぶ
	// フルページ URL は、リンク一覧が現在どこまで進んだかではなくそのページを描画する必要がある。
	// 描画対象にないカードを指すネストページ番号は、各画面が 404 で答えるためである。
	LinkedPageParentPage int32
}

// HasNestedBacklinkState reports whether the state names a card whose nested backlink list has been
// advanced past its first page. Below that the selection carries no information, because every card
// starts at its own first page.
//
// [Ja] HasNestedBacklinkState は、ネストしたバックリンク一覧を 2 ページ目以降へ進めたカードを状態が
// 指しているかを返す。それ未満では選択に情報が無い。どのカードも自分の 1 ページ目から始まるためである。
func (s PageLinkState) HasNestedBacklinkState() bool {
	return s.LinkedPageNumber > 0 && s.LinkedBacklinkPage > 1
}

// NestedBacklinkLinkPage returns the link-list page a request carrying the nested state must render
// for the selected card to be present. A state built before the parent page joined it falls back to
// the link list's own page, which is what such a URL meant at the time.
//
// Unlike the listing pages, the first page is a meaningful value here rather than a default worth
// omitting: a URL that drops it cannot be told from one built before the parent page existed, and
// would fall back to a link-list page the card is not on.
//
// [Ja] NestedBacklinkLinkPage は、ネスト状態を運ぶリクエストで選択カードが存在するために描画すべき
// リンク一覧のページを返す。親ページが状態に加わる前に組み立てられた状態はリンク一覧自身のページへ
// フォールバックする。当時の URL が意味していたのはその値だからである。
//
// 一覧のページ番号と違い、ここでの 1 ページ目は省略してよい既定値ではなく意味のある値である。省いた
// URL は親ページが存在しなかった頃の URL と区別できず、カードの載っていないリンク一覧ページへ
// フォールバックしてしまう。
func (s PageLinkState) NestedBacklinkLinkPage() int32 {
	if s.LinkedPageParentPage > 0 {
		return s.LinkedPageParentPage
	}

	return pageOrFirst(s.LinkPage)
}

// Normalized resolves the context and every page number to a value the rest of the screen can use
// as is: an unset or invalid page becomes the first page, and an unknown context becomes the
// editor. It is the one place the rule lives, so the slices a usecase fetches and the page numbers
// the listings render cannot drift apart. LinkedPageNumber keeps its zero value, which means no
// card is selected.
//
// [Ja] Normalized は文脈と各ページ番号を、画面の残りがそのまま使える値へ解決する。未設定・不正な
// ページは 1 ページ目に、未知の文脈は編集画面になる。この規則が存在する唯一の場所であり、UseCase が
// 取得する範囲と一覧が描画するページ番号がずれないようにする。LinkedPageNumber はゼロ値のままで、
// カードが選ばれていないことを表す。
func (s PageLinkState) Normalized() PageLinkState {
	s.Context = NormalizePageLinkContext(string(s.Context))
	s.LinkPage = pageOrFirst(s.LinkPage)
	s.LinkedBacklinkPage = pageOrFirst(s.LinkedBacklinkPage)
	s.PageBacklinkPage = pageOrFirst(s.PageBacklinkPage)

	return s
}

// WithinCumulativeLimit reports whether every listing page can be fetched cumulatively under the
// supplied server-side limit. A non-positive limit or the public-page context disables the bound
// for screens that fetch only one page at a time.
//
// [Ja] WithinCumulativeLimit は、各一覧のページが指定したサーバー側上限の範囲で累積取得できるかを
// 返す。0 以下の上限または公開ページの文脈では、常に 1 ページだけを取得する画面の制限を無効にする。
func (s PageLinkState) WithinCumulativeLimit(limit int32) bool {
	if limit <= 0 || !s.Context.IncludesPrecedingPages() {
		return true
	}

	s = s.Normalized()
	return s.LinkPage <= limit &&
		s.LinkedBacklinkPage <= limit &&
		s.PageBacklinkPage <= limit
}

// CumulativePageLimit returns the supplied limit only for the editor mode that rebuilds every page
// from the first through the current one. One-page-at-a-time contexts are unbounded by that prefix
// limit because each request still fetches a fixed-size slice.
//
// [Ja] CumulativePageLimit は、先頭から現在ページまでを毎回再構築する編集モードにだけ指定上限を返す。
// 1 ページ単位の文脈は各リクエストが固定長の範囲だけを取得するため、この先頭範囲上限を適用しない。
func (s PageLinkState) CumulativePageLimit(limit int32) int32 {
	if !s.Context.IncludesPrecedingPages() {
		return 0
	}

	return limit
}

// BuildPageLinkDataInput holds the input for BuildPageLinkData.
//
// [Ja] BuildPageLinkDataInput は BuildPageLinkData の入力パラメータ。
type BuildPageLinkDataInput struct {
	LinkedPages       []*model.Page
	LinkedTotalCount  int64
	BacklinksPerPage  map[model.PageID]*PageSliceWithCount
	PageBacklinks     []*model.Page
	PageBacklinkCount int64
	Topics            []*model.Topic
	SpaceIdentifier   model.SpaceIdentifier
	PageNumber        int32

	// LinkedPageFirstPage is the one-based page containing the first linked page in LinkedPages.
	// A cumulative draft refresh sets it to one, while one-page responses set it to State.LinkPage.
	// It lets a nested listing link back to the full-page editor slice containing its parent card.
	//
	// [Ja] LinkedPageFirstPage は LinkedPages の先頭要素を含む 1 始まりのページ番号。累積する下書き
	// 再取得では 1、1 ページ分の応答では State.LinkPage を設定する。ネスト一覧が親カードを含む編集画面の
	// ページへ戻るために使う。
	LinkedPageFirstPage int32

	// CumulativePageLimit stops the editor from offering a page its whole-list refresh cannot
	// fetch safely. Zero leaves the normal one-page-at-a-time pagination unbounded.
	//
	// [Ja] CumulativePageLimit は、一覧全体の再取得で安全に扱えないページを編集画面が提示しない
	// ようにする。0 の場合、通常の 1 ページ単位のページネーションには上限を設けない。
	CumulativePageLimit int32

	// State is the pagination state of all listings on the screen, which every "load more" link
	// carries so that advancing one listing leaves the others where they are.
	//
	// [Ja] State は画面上の全一覧のページネーション状態。各「もっと見る」リンクがこれを引き継ぐ
	// ことで、1 つの一覧を進めても他の一覧は現在位置のままになる。
	State PageLinkState

	// CanEdit turns the per-card edit link on. See NewLinkListInput.CanEdit.
	//
	// [Ja] CanEdit は各カードの編集リンクを出すかを表す (NewLinkListInput.CanEdit を参照)。
	CanEdit bool
}

// PageLinkData pairs the link list with the page's own backlink list.
//
// [Ja] PageLinkData はリンク一覧とページ自身のバックリンク一覧の ViewModel の組み合わせ。
type PageLinkData struct {
	LinkList     LinkList
	BacklinkList BacklinkList
}

// BacklinkPageFor returns the page of one listed page's nested backlink list. Only the card the
// state names is advanced; every other card keeps its first page, because the state points at a
// single card with one page number.
//
// [Ja] BacklinkPageFor は一覧した 1 ページ分のネストしたバックリンク一覧のページを返す。進めるのは
// 状態が指すカードだけで、他のカードは 1 ページ目のままになる。状態はページ番号 1 つで 1 枚のカード
// だけを指しているためである。
func (s PageLinkState) BacklinkPageFor(linkedPages []*model.Page, pageID model.PageID) int32 {
	for _, linkedPage := range linkedPages {
		if linkedPage.ID == pageID && int32(linkedPage.Number) == s.LinkedPageNumber {
			return pageOrFirst(s.LinkedBacklinkPage)
		}
	}

	return 1
}

// NewLinkedPageBacklinkListsInput holds the input for NewLinkedPageBacklinkLists.
//
// [Ja] NewLinkedPageBacklinkListsInput は NewLinkedPageBacklinkLists の入力パラメータ。
type NewLinkedPageBacklinkListsInput struct {
	LinkedPages         []*model.Page
	BacklinksPerPage    map[model.PageID]*PageSliceWithCount
	TopicMap            map[model.TopicID]*model.Topic
	SpaceIdentifier     model.SpaceIdentifier
	PageNumber          int32
	LinkedPageFirstPage int32
	CumulativePageLimit int32
	State               PageLinkState
	CanEdit             bool
}

// NewLinkedPageBacklinkLists builds the nested backlink list of every listed page, keyed by that
// page's ID. Each listing takes the number and title of the card it hangs off, which name it in the
// accessible name and the fallback URL of its "load more" link. The initial render and the fragment
// that continues the link list both need this lookup, so it lives here instead of in each caller.
//
// [Ja] NewLinkedPageBacklinkLists は一覧した各ページのネストしたバックリンク一覧を、そのページの ID
// をキーにして構築する。各一覧はぶら下がるカードの番号とタイトルを受け取り、それらが「もっと見る」
// リンクのアクセシブルネームとフォールバック URL でこの一覧を言い表す。初回描画と、リンク一覧の続きを
// 返すフラグメントの双方がこの逆引きを必要とするため、呼び出し元ごとに置かずここに置く。
func NewLinkedPageBacklinkLists(input NewLinkedPageBacklinkListsInput) map[model.PageID]BacklinkList {
	type linkedPageMeta struct {
		number     int32
		title      string
		parentPage int32
	}

	firstPage := pageOrFirst(input.LinkedPageFirstPage)
	metaByPageID := make(map[model.PageID]linkedPageMeta, len(input.LinkedPages))
	for index, linkedPage := range input.LinkedPages {
		title := ""
		if linkedPage.Title != nil {
			title = *linkedPage.Title
		}
		metaByPageID[linkedPage.ID] = linkedPageMeta{
			number:     int32(linkedPage.Number),
			title:      title,
			parentPage: RelatedPageForSliceIndex(firstPage, index, LinkLimit),
		}
	}

	lists := make(map[model.PageID]BacklinkList, len(input.BacklinksPerPage))
	for pageID, data := range input.BacklinksPerPage {
		meta := metaByPageID[pageID]
		pagination, capped := NewRelatedPagePagination(pageOrFirst(data.CurrentPage), data.TotalCount, BacklinkLimit, input.State, input.CumulativePageLimit)

		lists[pageID] = NewBacklinkList(NewBacklinkListInput{
			Pages:            data.Pages,
			TopicMap:         input.TopicMap,
			Pagination:       pagination,
			LoadMoreCapped:   capped,
			SpaceIdentifier:  input.SpaceIdentifier,
			PageNumber:       input.PageNumber,
			ParentLinkPage:   meta.parentPage,
			LinkedPageNumber: meta.number,
			LinkedPageTitle:  meta.title,
			State:            input.State,
			CanEdit:          input.CanEdit,
		})
	}

	return lists
}

// BuildPageLinkData builds the link list and the page's own backlink list. The editor and the page
// detail screen share it because they render the same two listing shapes.
//
// [Ja] BuildPageLinkData はリンク一覧とページ自身のバックリンク一覧の ViewModel を構築する。
// ページ編集画面とページ表示画面の双方が同じ 2 つの一覧を描画するため、両者で共有する。
func BuildPageLinkData(input BuildPageLinkDataInput) PageLinkData {
	topicMap := make(map[model.TopicID]*model.Topic, len(input.Topics))
	for _, t := range input.Topics {
		topicMap[t.ID] = t
	}

	state := input.State.Normalized()

	var linkListVM LinkList
	if len(input.LinkedPages) > 0 {
		backlinkMap := NewLinkedPageBacklinkLists(NewLinkedPageBacklinkListsInput{
			LinkedPages:         input.LinkedPages,
			BacklinksPerPage:    input.BacklinksPerPage,
			TopicMap:            topicMap,
			SpaceIdentifier:     input.SpaceIdentifier,
			PageNumber:          input.PageNumber,
			LinkedPageFirstPage: input.LinkedPageFirstPage,
			CumulativePageLimit: input.CumulativePageLimit,
			State:               state,
			CanEdit:             input.CanEdit,
		})

		linkPagination, linkCapped := NewRelatedPagePagination(state.LinkPage, input.LinkedTotalCount, LinkLimit, state, input.CumulativePageLimit)

		linkListVM = NewLinkList(NewLinkListInput{
			Pages:           input.LinkedPages,
			TopicMap:        topicMap,
			BacklinkMap:     backlinkMap,
			Pagination:      linkPagination,
			LoadMoreCapped:  linkCapped,
			SpaceIdentifier: input.SpaceIdentifier,
			PageNumber:      input.PageNumber,
			State:           state,
			CanEdit:         input.CanEdit,
		})
	}

	pageBacklinkPagination, pageBacklinkCapped := NewRelatedPagePagination(state.PageBacklinkPage, input.PageBacklinkCount, PageBacklinkLimit, state, input.CumulativePageLimit)

	backlinkListVM := NewBacklinkList(NewBacklinkListInput{
		Pages:           input.PageBacklinks,
		TopicMap:        topicMap,
		Pagination:      pageBacklinkPagination,
		LoadMoreCapped:  pageBacklinkCapped,
		SpaceIdentifier: input.SpaceIdentifier,
		PageNumber:      input.PageNumber,
		State:           state,
		CanEdit:         input.CanEdit,
	})

	return PageLinkData{
		LinkList:     linkListVM,
		BacklinkList: backlinkListVM,
	}
}

// RelatedPageTotalPages returns the number of pages for a listing whose first page contains
// initialLimit items and whose following pages contain initialLimit+1 items.
//
// [Ja] RelatedPageTotalPages は、初回ページに initialLimit 件、後続ページに initialLimit+1 件を
// 表示する一覧の総ページ数を返す。
func RelatedPageTotalPages(totalCount int64, initialLimit int32) int {
	if totalCount <= int64(initialLimit) || initialLimit <= 0 {
		return 1
	}

	followingLimit := int64(initialLimit) + 1
	remaining := totalCount - int64(initialLimit)
	total := 2 + (remaining-1)/followingLimit
	if total > math.MaxInt {
		return math.MaxInt
	}

	// #nosec G115 -- total is bounded to math.MaxInt immediately above.
	return int(total)
}

// RelatedPageNumberForIndex returns the one-based page containing a zero-based item index under
// the same first-page and following-page limits as RelatedPageTotalPages.
//
// [Ja] RelatedPageNumberForIndex は、RelatedPageTotalPages と同じ初回・後続ページ件数のもとで、
// 0 始まりの要素位置を含む 1 始まりのページ番号を返す。
func RelatedPageNumberForIndex(index int, initialLimit int32) int32 {
	if index < int(initialLimit) || initialLimit <= 0 {
		return 1
	}

	page := 2 + (int64(index)-int64(initialLimit))/(int64(initialLimit)+1)
	if page > math.MaxInt32 {
		return math.MaxInt32
	}

	// #nosec G115 -- page is positive and bounded to math.MaxInt32 immediately above.
	return int32(page)
}

// RelatedPageForSliceIndex maps an item in either a first-page/cumulative slice or a later
// single-page slice back to its parent page. firstPage is the listing page the slice starts at,
// which is one for a cumulative slice.
//
// The screens that rebuild a listing's shared pagination state after a draft change call it too, so
// that the page the state names and the page the cards were rendered from stay the same value.
//
// [Ja] RelatedPageForSliceIndex は、初回・累積範囲または後続の単一ページ範囲にある要素を、その親
// ページ番号へ対応付ける。firstPage は範囲の先頭にあたる一覧のページ番号で、累積範囲では 1 になる。
//
// 下書きの変更後に一覧の共有ページネーション状態を組み立て直す画面もこれを呼ぶ。状態が指すページと、
// カードを描画した元のページを同じ値に保つためである。
func RelatedPageForSliceIndex(firstPage int32, index int, initialLimit int32) int32 {
	if firstPage > 1 {
		return firstPage
	}

	return RelatedPageNumberForIndex(index, initialLimit)
}

// NewRelatedPagePagination builds pagination for a related-page listing and removes the editor's
// next-page affordance at the cumulative-fetch boundary. The first page contains initialLimit
// items and every following page contains initialLimit+1. The actual total remains intact for range
// checks, and public-page pagination remains unbounded.
//
// The second return value reports that a next page exists but is being withheld, which is what the
// listing renders its stop notice from. It stays false once the listing genuinely has no next page,
// so reaching the last item within the limit still ends the listing silently.
//
// [Ja] NewRelatedPagePagination は関連ページ一覧のページネーションを構築し、編集画面では累積取得
// 上限で次ページへの導線を止める。初回ページは initialLimit 件、後続ページは initialLimit+1 件に
// する。範囲チェックに使う実際の総ページ数は維持し、公開ページの
// ページネーションには上限を設けない。
//
// 2 つ目の返り値は、次ページが存在するのに出していないことを表し、一覧はこれを見て打ち切りの案内を
// 描画する。一覧に本当に次ページが無い場合は false のままで、上限内で最後まで到達したときは従来どおり
// 案内なしで終わる。
func NewRelatedPagePagination(current int32, totalCount int64, initialLimit int32, state PageLinkState, cumulativePageLimit int32) (Pagination, bool) {
	total := RelatedPageTotalPages(totalCount, initialLimit)
	pagination := Pagination{
		Current:     int(current),
		Total:       total,
		HasNext:     int(current) < total,
		HasPrevious: current > 1,
	}

	capped := state.Context.IncludesPrecedingPages() &&
		cumulativePageLimit > 0 &&
		current >= cumulativePageLimit &&
		pagination.HasNext
	if capped {
		pagination.HasNext = false
	}

	return pagination, capped
}

// pageOrFirst resolves an unset or invalid page number to the first page, so that a caller that
// does not paginate a listing can leave the field at its zero value. PageLinkState.Normalized and
// the per-card page of PageSliceWithCount share it, which keeps one rule behind both.
//
// [Ja] pageOrFirst は未設定・不正なページ番号を 1 ページ目へ解決する。一覧をページングしない
// 呼び出し元がフィールドをゼロ値のままにできるようにするためである。PageLinkState.Normalized と
// PageSliceWithCount のカードごとのページが共有し、両者の背後の規則を 1 つに保つ。
func pageOrFirst(page int32) int32 {
	if page > 0 {
		return page
	}
	return 1
}
