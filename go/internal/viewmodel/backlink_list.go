package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// BacklinkListItemはバックリンクの個別項目です
type BacklinkListItem struct {
	CardLinkPage CardLinkPage
}

// BacklinkListはバックリンク一覧の表示データです
type BacklinkList struct {
	Items           []BacklinkListItem
	Pagination      Pagination
	SpaceIdentifier SpaceIdentifier
	PageNumber      int32

	// ParentLinkPageは、このネスト一覧が属するカードを含むリンク一覧のページ番号。ページ自身の
	// バックリンクではゼロで、上限到達後の編集画面フォールバックが親カードを描画するために使う。
	ParentLinkPage int32

	// LoadMoreCappedは、次ページが存在するのに編集画面がそれを出していないことを表す
	// (理由はLinkList.LoadMoreCappedと同じ)。
	LoadMoreCapped bool

	// LinkedPageNumberとLinkedPageTitleは、このバックリンク一覧がぶら下がるリンク先ページを
	// 表し、ページ自身のバックリンク一覧ではゼロ値になる。タイトルは「もっと見る」リンクの
	// アクセシブルネームでこの一覧を言い表すために使う。
	LinkedPageNumber int32
	LinkedPageTitle  string

	// Stateは画面上の全一覧のページネーション状態 (PageLinkStateを参照)。
	State PageLinkState
}

// NewBacklinkListInputはNewBacklinkListの入力パラメータです
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

	// CanEditは各カードの編集リンクを出すかを表す (理由はNewLinkListInput.CanEditと同じ)。
	CanEdit bool
}

// NewBacklinkListはバックリンクページの一覧からBacklinkListを生成します
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
