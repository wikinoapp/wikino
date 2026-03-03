package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

const (
	// LinkLimit はリンク一覧の1ページあたりの表示件数です
	LinkLimit int32 = 15
	// BacklinkLimit はバックリンクの1ページあたりの表示件数です
	BacklinkLimit int32 = 13
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
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
}

// NewLinkListInput はNewLinkListの入力パラメータです
type NewLinkListInput struct {
	Pages           []*model.Page
	TopicMap        map[model.TopicID]*model.Topic
	BacklinkMap     map[model.PageID]BacklinkList
	Pagination      Pagination
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
}

// NewLinkList はリンク先ページの一覧からLinkListを生成します
func NewLinkList(input NewLinkListInput) LinkList {
	items := make([]LinkListItem, 0, len(input.Pages))
	for _, pg := range input.Pages {
		item := LinkListItem{
			CardLinkPage: NewCardLinkPage(pg, input.TopicMap),
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
		PageNumber:      input.PageNumber,
	}
}
