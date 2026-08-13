package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

const (
	// LinkLimit is the number of linked pages rendered per page.
	// [Ja] LinkLimit はリンク一覧の 1 ページあたりの表示件数。
	LinkLimit int32 = 15
	// BacklinkLimit is the number of nested backlinks rendered per page.
	// [Ja] BacklinkLimit はネストしたバックリンクの 1 ページあたりの表示件数。
	BacklinkLimit int32 = 13
	// PageBacklinkLimit is the number of page-level backlinks rendered per page.
	// [Ja] PageBacklinkLimit はページ自身のバックリンク一覧の 1 ページあたりの表示件数。
	PageBacklinkLimit int32 = 14
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
		card.Primary = true
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
