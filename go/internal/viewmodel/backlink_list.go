package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// BacklinkListItem はバックリンクの個別項目です
type BacklinkListItem struct {
	CardLinkPage CardLinkPage
}

// BacklinkList はバックリンク一覧の表示データです
type BacklinkList struct {
	Items           []BacklinkListItem
	Pagination      Pagination
	SpaceIdentifier SpaceIdentifier
	PageNumber      int32

	// ParentLinkPage is the link-list page containing the card this nested list belongs to. It is
	// zero for the page's own backlinks and lets the capped editor fallback render the parent card.
	//
	// [Ja] ParentLinkPage は、このネスト一覧が属するカードを含むリンク一覧のページ番号。ページ自身の
	// バックリンクではゼロで、上限到達後の編集画面フォールバックが親カードを描画するために使う。
	ParentLinkPage int32

	// LoadMoreCapped reports that a next page exists but the editor withholds it, for the same
	// reason as LinkList.LoadMoreCapped.
	//
	// [Ja] LoadMoreCapped は、次ページが存在するのに編集画面がそれを出していないことを表す
	// (理由は LinkList.LoadMoreCapped と同じ)。
	LoadMoreCapped bool

	// LinkedPageNumber and LinkedPageTitle identify the listed page this backlink list hangs off,
	// and are zero for the page's own backlink list. The title names the listing in the accessible
	// name of its "load more" link.
	//
	// [Ja] LinkedPageNumber と LinkedPageTitle は、このバックリンク一覧がぶら下がるリンク先ページを
	// 表し、ページ自身のバックリンク一覧ではゼロ値になる。タイトルは「もっと見る」リンクの
	// アクセシブルネームでこの一覧を言い表すために使う。
	LinkedPageNumber int32
	LinkedPageTitle  string

	// State is the pagination state of every listing on the screen (see PageLinkState).
	//
	// [Ja] State は画面上の全一覧のページネーション状態 (PageLinkState を参照)。
	State PageLinkState
}

// NewBacklinkListInput はNewBacklinkListの入力パラメータです
type NewBacklinkListInput struct {
	Pages            []*model.Page
	TopicMap         map[model.TopicID]*model.Topic
	Pagination       Pagination
	LoadMoreCapped   bool
	SpaceIdentifier  model.SpaceIdentifier
	PageNumber       int32
	ParentLinkPage   int32
	LinkedPageNumber int32
	LinkedPageTitle  string
	State            PageLinkState

	// CanEdit turns the per-card edit link on, for the same reason as NewLinkListInput.CanEdit.
	//
	// [Ja] CanEdit は各カードの編集リンクを出すかを表す (理由は NewLinkListInput.CanEdit と同じ)。
	CanEdit bool
}

// NewBacklinkList はバックリンクページの一覧からBacklinkListを生成します
func NewBacklinkList(input NewBacklinkListInput) BacklinkList {
	items := make([]BacklinkListItem, 0, len(input.Pages))
	for _, pg := range input.Pages {
		card := NewCardLinkPage(pg, input.TopicMap)
		card.CanEdit = input.CanEdit
		items = append(items, BacklinkListItem{
			CardLinkPage: card,
		})
	}
	return BacklinkList{
		Items:            items,
		Pagination:       input.Pagination,
		LoadMoreCapped:   input.LoadMoreCapped,
		SpaceIdentifier:  NewSpaceIdentifier(input.SpaceIdentifier),
		PageNumber:       input.PageNumber,
		ParentLinkPage:   input.ParentLinkPage,
		LinkedPageNumber: input.LinkedPageNumber,
		LinkedPageTitle:  input.LinkedPageTitle,
		State:            input.State,
	}
}
