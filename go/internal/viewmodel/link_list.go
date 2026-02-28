package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// LinkListItem はリンク一覧の個別リンク情報です
type LinkListItem struct {
	Page         Page
	BacklinkList BacklinkList
}

// LinkList はリンク一覧の表示データです
type LinkList struct {
	Items           []LinkListItem
	Pagination      Pagination
	SpaceIdentifier string
}

// NewLinkListInput はNewLinkListの入力パラメータです
type NewLinkListInput struct {
	Pages           []*model.Page
	BacklinkMap     map[model.PageID]BacklinkList
	Pagination      Pagination
	SpaceIdentifier string
}

// NewLinkList はリンク先ページの一覧からLinkListを生成します
func NewLinkList(input NewLinkListInput) LinkList {
	items := make([]LinkListItem, 0, len(input.Pages))
	for _, pg := range input.Pages {
		item := LinkListItem{
			Page: newPageFromModel(pg),
		}
		if input.BacklinkMap != nil {
			item.BacklinkList = input.BacklinkMap[pg.ID]
		}
		items = append(items, item)
	}
	return LinkList{
		Items:           items,
		Pagination:      input.Pagination,
		SpaceIdentifier: input.SpaceIdentifier,
	}
}
