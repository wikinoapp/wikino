package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// LinkListItem はリンク一覧の個別リンク情報です
type LinkListItem struct {
	Title  string
	Number int32
}

// LinkList はリンク一覧の表示データです
type LinkList struct {
	Items           []LinkListItem
	SpaceIdentifier string
}

// NewLinkList はリンク先ページの一覧からLinkListを生成します
func NewLinkList(pages []*model.Page, spaceIdentifier string) LinkList {
	items := make([]LinkListItem, 0, len(pages))
	for _, pg := range pages {
		var title string
		if pg.Title != nil {
			title = *pg.Title
		}
		items = append(items, LinkListItem{
			Title:  title,
			Number: int32(pg.Number),
		})
	}
	return LinkList{
		Items:           items,
		SpaceIdentifier: spaceIdentifier,
	}
}
