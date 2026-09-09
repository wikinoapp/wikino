package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// Card counts of the three related-page listings. Each listing is laid out in a grid that is one
// column on a narrow screen and three columns from md up, and its "load more" tile follows the
// cards as one more cell of that same grid. The tile ends a complete row while the loaded card
// count leaves a remainder of two on division by three: the initial page satisfies it, and every
// following page appends a multiple of three, so it survives each append. A single column puts
// every cell at both ends, so the narrow layout holds for either count.
//
// The other counts are derived from RelatedPageInitialLimit, because the pagination that resolves
// them (RelatedPageTotalPages here, listingWindow in the usecase package) computes a following page
// as one card more than the initial one. Writing the following count as its own literal would let
// the two say different things, and the offset bound the handlers check would then no longer be the
// largest page a listing can return.
//
// [Ja] 3 つの関連ページ一覧のカード数。各一覧は狭い画面では 1 カラム、md 以上では 3 カラムの
// グリッドに並び、「もっと見る」のタイルはカードに続く同じグリッドの 1 マスとして置かれる。読み込み
// 済みカード数を 3 で割った余りが 2 である間、タイルは埋まった行の末尾に来る。初回ページがこれを
// 満たし、各後続ページは 3 の倍数を追記するため、追記後も保たれる。1 カラムではどのマスも両端に
// 来るため、どちらの件数でも狭い画面の並びは成り立つ。
//
// 他の件数はすべて RelatedPageInitialLimit から導出する。これらを解決するページネーション
// (本パッケージの RelatedPageTotalPages と usecase パッケージの listingWindow) が、後続ページを
// 初回より 1 件多いものとして計算するためである。後続の件数を別のリテラルで書くと両者が食い違い、
// Handler が検査する offset の上限が、一覧の返しうる最大ページ件数でなくなってしまう。
const (
	// RelatedPageInitialLimit is the number of cards the first page of every related-page listing
	// holds.
	//
	// [Ja] RelatedPageInitialLimit は、各関連ページ一覧の 1 ページ目が持つカード数。
	RelatedPageInitialLimit int32 = 14

	// RelatedPageFollowingLimit is the number of cards appended by every following page.
	//
	// [Ja] RelatedPageFollowingLimit は各後続ページで追記するカード数。
	RelatedPageFollowingLimit int32 = RelatedPageInitialLimit + 1

	// LinkLimit is the number of linked pages rendered initially.
	//
	// [Ja] LinkLimit はリンク一覧の初回表示件数。
	LinkLimit int32 = RelatedPageInitialLimit

	// BacklinkLimit is the number of nested backlinks rendered initially.
	//
	// [Ja] BacklinkLimit はネストしたバックリンクの初回表示件数。
	BacklinkLimit int32 = RelatedPageInitialLimit

	// PageBacklinkLimit is the number of page-level backlinks rendered initially.
	//
	// [Ja] PageBacklinkLimit はページ自身のバックリンク一覧の初回表示件数。
	PageBacklinkLimit int32 = RelatedPageInitialLimit
)

// LinkListItem はリンク一覧の個別リンク情報です
type LinkListItem struct {
	CardLinkPage CardLinkPage
	BacklinkList BacklinkList
}

// LinkList はリンク一覧の表示データです
type LinkList struct {
	Items           []LinkListItem
	Pagination      Pagination
	SpaceIdentifier SpaceIdentifier
	PageNumber      int32

	// LoadMoreCapped reports that a next page exists but the editor withholds it (see
	// NewRelatedPagePagination), so the listing says where it stops instead of ending silently.
	//
	// [Ja] LoadMoreCapped は、次ページが存在するのに編集画面がそれを出していないことを表す
	// (NewRelatedPagePagination を参照)。一覧は黙って終わるのではなく、どこで止まったかを伝える。
	LoadMoreCapped bool

	// State is the pagination state of every listing on the screen (see PageLinkState).
	//
	// [Ja] State は画面上の全一覧のページネーション状態 (PageLinkState を参照)。
	State PageLinkState
}

// NewLinkListInput はNewLinkListの入力パラメータです
type NewLinkListInput struct {
	Pages           []*model.Page
	TopicMap        map[model.TopicID]*model.Topic
	BacklinkMap     map[model.PageID]BacklinkList
	Pagination      Pagination
	LoadMoreCapped  bool
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	State           PageLinkState

	// CanEdit turns the per-card edit link on. The listing is shown on the public page detail
	// screen as well, where a guest must not be offered an edit link, so the caller passes the
	// viewer's own permission instead of it being hard-coded here.
	//
	// [Ja] CanEdit は各カードの編集リンクを出すかを表す。この一覧は公開のページ表示画面にも出るため、
	// ゲストに編集リンクを見せないよう、ここで固定せず呼び出し元が閲覧者の権限を渡す。
	CanEdit bool
}

// NewLinkList はリンク先ページの一覧からLinkListを生成します
func NewLinkList(input NewLinkListInput) LinkList {
	items := make([]LinkListItem, 0, len(input.Pages))
	for _, pg := range input.Pages {
		card := NewCardLinkPage(pg, input.TopicMap)
		card.CanEdit = input.CanEdit
		item := LinkListItem{
			CardLinkPage: card,
		}
		if input.BacklinkMap != nil {
			item.BacklinkList = input.BacklinkMap[pg.ID]
		}
		items = append(items, item)
	}
	return LinkList{
		Items:           items,
		Pagination:      input.Pagination,
		LoadMoreCapped:  input.LoadMoreCapped,
		SpaceIdentifier: NewSpaceIdentifier(input.SpaceIdentifier),
		PageNumber:      input.PageNumber,
		State:           input.State,
	}
}
