package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// 3つの関連ページ一覧のカード数。各一覧は狭い画面では1カラム、md以上では3カラムの
// グリッドに並び、「もっと見る」のタイルはカードに続く同じグリッドの1マスとして置かれる。読み込み
// 済みカード数を3で割った余りが2である間、タイルは埋まった行の末尾に来る。初回ページがこれを
// 満たし、各後続ページは3の倍数を追記するため、追記後も保たれる。1カラムではどのマスも両端に
// 来るため、どちらの件数でも狭い画面の並びは成り立つ。
//
// 他の件数はすべてRelatedPageInitialLimitから導出する。これらを解決するページネーション
// (本パッケージのRelatedPageTotalPagesとusecaseパッケージのlistingWindow) が、後続ページを
// 初回より1件多いものとして計算するためである。後続の件数を別のリテラルで書くと両者が食い違い、
// Handlerが検査するoffsetの上限が、一覧の返しうる最大ページ件数でなくなってしまう。
const (
	// RelatedPageInitialLimitは、各関連ページ一覧の1ページ目が持つカード数。
	RelatedPageInitialLimit int32 = 14

	// RelatedPageFollowingLimitは各後続ページで追記するカード数。
	RelatedPageFollowingLimit int32 = RelatedPageInitialLimit + 1

	// LinkLimitはリンク一覧の初回表示件数。
	LinkLimit int32 = RelatedPageInitialLimit

	// BacklinkLimitはネストしたバックリンクの初回表示件数。
	BacklinkLimit int32 = RelatedPageInitialLimit

	// PageBacklinkLimitはページ自身のバックリンク一覧の初回表示件数。
	PageBacklinkLimit int32 = RelatedPageInitialLimit
)

// LinkListItemはリンク一覧の個別リンク情報です
type LinkListItem struct {
	CardLinkPage CardLinkPage
	BacklinkList BacklinkList
}

// LinkListはリンク一覧の表示データです
type LinkList struct {
	Items           []LinkListItem
	Pagination      Pagination
	SpaceIdentifier SpaceIdentifier
	PageNumber      int32

	// LoadMoreCappedは、次ページが存在するのに編集画面がそれを出していないことを表す
	// (NewRelatedPagePaginationを参照)。一覧は黙って終わるのではなく、どこで止まったかを伝える。
	LoadMoreCapped bool

	// Stateは画面上の全一覧のページネーション状態 (PageLinkStateを参照)。
	State PageLinkState
}

// NewLinkListInputはNewLinkListの入力パラメータです
type NewLinkListInput struct {
	Pages           []*model.Page
	TopicMap        map[model.TopicID]*model.Topic
	BacklinkMap     map[model.PageID]BacklinkList
	Pagination      Pagination
	LoadMoreCapped  bool
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	State           PageLinkState

	// CanEditは各カードの編集リンクを出すかを表す。この一覧は公開のページ表示画面にも出るため、
	// ゲストに編集リンクを見せないよう、ここで固定せず呼び出し元が閲覧者の権限を渡す。
	CanEdit bool
}

// NewLinkListはリンク先ページの一覧からLinkListを生成します
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
