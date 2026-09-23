package viewmodel

import (
	"math"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// PageSliceWithCountは関連ページ一覧の1範囲と総件数・現在ページを組み合わせる。
type PageSliceWithCount struct {
	Pages       []*model.Page
	TotalCount  int64
	CurrentPage int32
}

// PageLinkContextは関連ページ一覧を持つフルページを表す。フラグメントURLがこの値を
// 引き継ぐことで、プログレッシブエンハンスメントされたリンクは正しい親画面へフォールバックできる。
type PageLinkContext string

const (
	// PageLinkContextEditはメンバーの下書きをリンク一覧へ反映するページ編集画面。
	PageLinkContextEdit PageLinkContext = "edit"

	// PageLinkContextEditPaginatedは、累積一覧が安全上限へ達したあとのページ編集画面。各関連
	// ページ一覧を要求された1ページずつ描画し、下書き再取得に無制限な先頭範囲を取得させずに、
	// メンバーが下書き固有のリンクを辿り続けられるようにする。
	PageLinkContextEditPaginated PageLinkContext = "edit_paginated"

	// PageLinkContextShowは保存済みリンクだけを使う公開ページ表示画面。
	PageLinkContextShow PageLinkContext = "show"
)

// 関連ページ一覧で共有するクエリパラメータ名。一覧を描画する画面と、その続きを返す
// フラグメントエンドポイントの双方がこれを参照するため、リネームしても両者がずれない。
const (
	// PageLinkContextQueryParamは画面からフラグメントへPageLinkContextを伝える。
	PageLinkContextQueryParam = "context"

	// LinkPageQueryParamはフルページリクエストでリンク一覧のページを選ぶ。
	LinkPageQueryParam = "links_page"

	// LinkedPageNumberQueryParamは、どのリンク先ページのネストしたバックリンク一覧を
	// 進めるかを選ぶ。
	LinkedPageNumberQueryParam = "linked_page_number"

	// LinkedBacklinkPageQueryParamはそのネストしたバックリンク一覧のページを選ぶ。
	LinkedBacklinkPageQueryParam = "linked_backlinks_page"

	// LinkedPageParentPageQueryParamは選択したカードを含むリンク一覧のページ番号を運ぶ。この値は
	// そのカードのネストした一覧に属するが、編集画面のリクエストでは画面全体の共有状態のうち
	// ネスト部分として運ぶ。
	LinkedPageParentPageQueryParam = "linked_page_parent_page"

	// FragmentParentPageQueryParamはLinkedPageParentPageQueryParamと同種の値を運ぶが、指すのは
	// 画面が現在開いているカードではなく、ネストしたバックリンクのフラグメントが進めるカードである。
	// 名前を分ける必要があるのは、編集画面のリクエストが画面全体の状態も送っており、htmxが同名の
	// 送信値でクエリパラメータを置き換えるためである。名前を共有すると、進めるカード自身のページが
	// 失われる。フラグメント自身の "page" を一覧ごとのページ名と分けているのと同じ切り分けである。
	FragmentParentPageQueryParam = "parent_page"

	// PageBacklinkPageQueryParamはページ自身のバックリンク一覧のページを選ぶ。
	PageBacklinkPageQueryParam = "backlinks_page"
)

// NormalizePageLinkContextは生のクエリ値を文脈へ解決する。ページ表示画面の値以外はすべて
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

// IsEditは、編集画面のどちらかのページング方式に属する文脈かを返す。
func (c PageLinkContext) IsEdit() bool {
	return NormalizePageLinkContext(string(c)) != PageLinkContextShow
}

// IncludesPrecedingPagesは、下書き再取得が読み込み済みの先頭範囲全体を再構築する必要があるかを
// 返す。ページ単位の編集画面と公開ページは、要求された1ページだけを差し替える。
func (c PageLinkContext) IncludesPrecedingPages() bool {
	return NormalizePageLinkContext(string(c)) == PageLinkContextEdit
}

// PageLinkStateは1画面上の関連ページ一覧すべてのページネーション状態。各「もっと見る」
// リンクは1つの一覧だけを進め、独立した一覧の状態はそのまま引き継ぐ。htmxで親のリンク一覧を
// 追記するときは既存のカードが残るためネストしたバックリンク状態を維持し、親ページを置き換える
// フルページフォールバックでは、入れ替わるカードに従属する子状態をリセットする。
//
// ネストしたバックリンク一覧を進められるのは一度に1つのリンク先ページだけである。状態が
// それをページ番号1つで指しているためである。
type PageLinkState struct {
	Context            PageLinkContext
	LinkPage           int32
	LinkedPageNumber   int32
	LinkedBacklinkPage int32
	PageBacklinkPage   int32

	// LinkedPageParentPageはLinkedPageNumberが指すカードを含むリンク一覧のページ。カードが
	// 選ばれていないときはゼロになる。カードは描画元のページにしか存在しないため、ネスト状態を運ぶ
	// フルページURLは、リンク一覧が現在どこまで進んだかではなくそのページを描画する必要がある。
	// 描画対象にないカードを指すネストページ番号は、各画面が404で答えるためである。
	LinkedPageParentPage int32
}

// HasNestedBacklinkStateは、ネストしたバックリンク一覧を2ページ目以降へ進めたカードを状態が
// 指しているかを返す。それ未満では選択に情報が無い。どのカードも自分の1ページ目から始まるためである。
func (s PageLinkState) HasNestedBacklinkState() bool {
	return s.LinkedPageNumber > 0 && s.LinkedBacklinkPage > 1
}

// NestedBacklinkLinkPageは、ネスト状態を運ぶリクエストで選択カードが存在するために描画すべき
// リンク一覧のページを返す。親ページが状態に加わる前に組み立てられた状態はリンク一覧自身のページへ
// フォールバックする。当時のURLが意味していたのはその値だからである。
//
// 一覧のページ番号と違い、ここでの1ページ目は省略してよい既定値ではなく意味のある値である。省いた
// URLは親ページが存在しなかった頃のURLと区別できず、カードの載っていないリンク一覧ページへ
// フォールバックしてしまう。
func (s PageLinkState) NestedBacklinkLinkPage() int32 {
	if s.LinkedPageParentPage > 0 {
		return s.LinkedPageParentPage
	}

	return pageOrFirst(s.LinkPage)
}

// Normalizedは文脈と各ページ番号を、画面の残りがそのまま使える値へ解決する。未設定・不正な
// ページは1ページ目に、未知の文脈は編集画面になる。この規則が存在する唯一の場所であり、UseCaseが
// 取得する範囲と一覧が描画するページ番号がずれないようにする。LinkedPageNumberはゼロ値のままで、
// カードが選ばれていないことを表す。
func (s PageLinkState) Normalized() PageLinkState {
	s.Context = NormalizePageLinkContext(string(s.Context))
	s.LinkPage = pageOrFirst(s.LinkPage)
	s.LinkedBacklinkPage = pageOrFirst(s.LinkedBacklinkPage)
	s.PageBacklinkPage = pageOrFirst(s.PageBacklinkPage)

	return s
}

// WithinCumulativeLimitは、各一覧のページが指定したサーバー側上限の範囲で累積取得できるかを
// 返す。0以下の上限または公開ページの文脈では、常に1ページだけを取得する画面の制限を無効にする。
func (s PageLinkState) WithinCumulativeLimit(limit int32) bool {
	if limit <= 0 || !s.Context.IncludesPrecedingPages() {
		return true
	}

	s = s.Normalized()
	return s.LinkPage <= limit &&
		s.LinkedBacklinkPage <= limit &&
		s.PageBacklinkPage <= limit
}

// CumulativePageLimitは、先頭から現在ページまでを毎回再構築する編集モードにだけ指定上限を返す。
// 1ページ単位の文脈は各リクエストが固定長の範囲だけを取得するため、この先頭範囲上限を適用しない。
func (s PageLinkState) CumulativePageLimit(limit int32) int32 {
	if !s.Context.IncludesPrecedingPages() {
		return 0
	}

	return limit
}

// BuildPageLinkDataInputはBuildPageLinkDataの入力パラメータ。
type BuildPageLinkDataInput struct {
	LinkedPages       []*model.Page
	LinkedTotalCount  int64
	BacklinksPerPage  map[model.PageID]*PageSliceWithCount
	PageBacklinks     []*model.Page
	PageBacklinkCount int64
	Topics            []*model.Topic
	SpaceIdentifier   model.SpaceIdentifier
	PageNumber        int32

	// LinkedPageFirstPageはLinkedPagesの先頭要素を含む1始まりのページ番号。累積する下書き
	// 再取得では1、1ページ分の応答ではState.LinkPageを設定する。ネスト一覧が親カードを含む編集画面の
	// ページへ戻るために使う。
	LinkedPageFirstPage int32

	// CumulativePageLimitは、一覧全体の再取得で安全に扱えないページを編集画面が提示しない
	// ようにする。0の場合、通常の1ページ単位のページネーションには上限を設けない。
	CumulativePageLimit int32

	// Stateは画面上の全一覧のページネーション状態。各「もっと見る」リンクがこれを引き継ぐ
	// ことで、1つの一覧を進めても他の一覧は現在位置のままになる。
	State PageLinkState

	// CanEditは各カードの編集リンクを出すかを表す (NewLinkListInput.CanEditを参照)。
	CanEdit bool
}

// PageLinkDataはリンク一覧とページ自身のバックリンク一覧のViewModelの組み合わせ。
type PageLinkData struct {
	LinkList     LinkList
	BacklinkList BacklinkList
}

// BacklinkPageForは一覧した1ページ分のネストしたバックリンク一覧のページを返す。進めるのは
// 状態が指すカードだけで、他のカードは1ページ目のままになる。状態はページ番号1つで1枚のカード
// だけを指しているためである。
func (s PageLinkState) BacklinkPageFor(linkedPages []*model.Page, pageID model.PageID) int32 {
	for _, linkedPage := range linkedPages {
		if linkedPage.ID == pageID && int32(linkedPage.Number) == s.LinkedPageNumber {
			return pageOrFirst(s.LinkedBacklinkPage)
		}
	}

	return 1
}

// NewLinkedPageBacklinkListsInputはNewLinkedPageBacklinkListsの入力パラメータ。
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

// NewLinkedPageBacklinkListsは一覧した各ページのネストしたバックリンク一覧を、そのページのID
// をキーにして構築する。各一覧はぶら下がるカードの番号とタイトルを受け取り、それらが「もっと見る」
// リンクのアクセシブルネームとフォールバックURLでこの一覧を言い表す。初回描画と、リンク一覧の続きを
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

// BuildPageLinkDataはリンク一覧とページ自身のバックリンク一覧のViewModelを構築する。
// ページ編集画面とページ表示画面の双方が同じ2つの一覧を描画するため、両者で共有する。
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

// RelatedPageTotalPagesは、初回ページにinitialLimit件、後続ページにinitialLimit+1件を
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

	// #nosec G115 -- totalは直前でmath.MaxInt以下に制限済み。
	return int(total)
}

// RelatedPageNumberForIndexは、RelatedPageTotalPagesと同じ初回・後続ページ件数のもとで、
// 0始まりの要素位置を含む1始まりのページ番号を返す。
func RelatedPageNumberForIndex(index int, initialLimit int32) int32 {
	if index < int(initialLimit) || initialLimit <= 0 {
		return 1
	}

	page := 2 + (int64(index)-int64(initialLimit))/(int64(initialLimit)+1)
	if page > math.MaxInt32 {
		return math.MaxInt32
	}

	// #nosec G115 -- pageは正で、直前でmath.MaxInt32以下に制限済み。
	return int32(page)
}

// RelatedPageForSliceIndexは、初回・累積範囲または後続の単一ページ範囲にある要素を、その親
// ページ番号へ対応付ける。firstPageは範囲の先頭にあたる一覧のページ番号で、累積範囲では1になる。
//
// 下書きの変更後に一覧の共有ページネーション状態を組み立て直す画面もこれを呼ぶ。状態が指すページと、
// カードを描画した元のページを同じ値に保つためである。
func RelatedPageForSliceIndex(firstPage int32, index int, initialLimit int32) int32 {
	if firstPage > 1 {
		return firstPage
	}

	return RelatedPageNumberForIndex(index, initialLimit)
}

// NewRelatedPagePaginationは関連ページ一覧のページネーションを構築し、編集画面では累積取得
// 上限で次ページへの導線を止める。初回ページはinitialLimit件、後続ページはinitialLimit+1件に
// する。範囲チェックに使う実際の総ページ数は維持し、公開ページの
// ページネーションには上限を設けない。
//
// 2つ目の返り値は、次ページが存在するのに出していないことを表し、一覧はこれを見て打ち切りの案内を
// 描画する。一覧に本当に次ページが無い場合はfalseのままで、上限内で最後まで到達したときは従来どおり
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

// pageOrFirstは未設定・不正なページ番号を1ページ目へ解決する。一覧をページングしない
// 呼び出し元がフィールドをゼロ値のままにできるようにするためである。PageLinkState.Normalizedと
// PageSliceWithCountのカードごとのページが共有し、両者の背後の規則を1つに保つ。
func pageOrFirst(page int32) int32 {
	if page > 0 {
		return page
	}
	return 1
}
