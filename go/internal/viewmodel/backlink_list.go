package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// BacklinkListItem はバックリンクの個別項目です
type BacklinkListItem struct {
	Page Page
}

// BacklinkList はバックリンク一覧の表示データです
type BacklinkList struct {
	Items            []BacklinkListItem
	Pagination       Pagination
	SpaceIdentifier  string
	PageNumber       int32
	LinkedPageNumber int32
}

// NewBacklinkListInput はNewBacklinkListの入力パラメータです
type NewBacklinkListInput struct {
	Pages            []*model.Page
	Pagination       Pagination
	SpaceIdentifier  string
	PageNumber       int32
	LinkedPageNumber int32
}

// NewBacklinkList はバックリンクページの一覧からBacklinkListを生成します
func NewBacklinkList(input NewBacklinkListInput) BacklinkList {
	items := make([]BacklinkListItem, 0, len(input.Pages))
	for _, pg := range input.Pages {
		items = append(items, BacklinkListItem{
			Page: newPageFromModel(pg),
		})
	}
	return BacklinkList{
		Items:            items,
		Pagination:       input.Pagination,
		SpaceIdentifier:  input.SpaceIdentifier,
		PageNumber:       input.PageNumber,
		LinkedPageNumber: input.LinkedPageNumber,
	}
}
