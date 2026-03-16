package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// PageSliceWithCount はページスライスと総件数のペアです
type PageSliceWithCount struct {
	Pages      []*model.Page
	TotalCount int64
}

// BuildEditLinkDataInput は BuildEditLinkData の入力パラメータです
type BuildEditLinkDataInput struct {
	LinkedPages       []*model.Page
	LinkedTotalCount  int64
	BacklinksPerPage  map[model.PageID]*PageSliceWithCount
	PageBacklinks     []*model.Page
	PageBacklinkCount int64
	Topics            []*model.Topic
	SpaceIdentifier   model.SpaceIdentifier
	PageNumber        int32
	CurrentPage       int32
}

// EditLinkData はリンク一覧・バックリンク一覧のViewModelの組み合わせです
type EditLinkData struct {
	LinkList     LinkList
	BacklinkList BacklinkList
}

// BuildEditLinkData はリンク一覧・バックリンク一覧のViewModelを構築します
func BuildEditLinkData(input BuildEditLinkDataInput) EditLinkData {
	topicMap := make(map[model.TopicID]*model.Topic, len(input.Topics))
	for _, t := range input.Topics {
		topicMap[t.ID] = t
	}

	var linkListVM LinkList
	if len(input.LinkedPages) > 0 {
		backlinkMap := make(map[model.PageID]BacklinkList, len(input.BacklinksPerPage))
		for pageID, data := range input.BacklinksPerPage {
			var linkedPageNumber int32
			for _, p := range input.LinkedPages {
				if p.ID == pageID {
					linkedPageNumber = int32(p.Number)
					break
				}
			}
			backlinkMap[pageID] = NewBacklinkList(NewBacklinkListInput{
				Pages:            data.Pages,
				TopicMap:         topicMap,
				Pagination:       NewPagination(1, data.TotalCount, int(BacklinkLimit)),
				SpaceIdentifier:  input.SpaceIdentifier,
				PageNumber:       input.PageNumber,
				LinkedPageNumber: linkedPageNumber,
			})
		}

		linkListVM = NewLinkList(NewLinkListInput{
			Pages:           input.LinkedPages,
			TopicMap:        topicMap,
			BacklinkMap:     backlinkMap,
			Pagination:      NewPagination(int(input.CurrentPage), input.LinkedTotalCount, int(LinkLimit)),
			SpaceIdentifier: input.SpaceIdentifier,
			PageNumber:      input.PageNumber,
		})
	}

	backlinkListVM := NewBacklinkList(NewBacklinkListInput{
		Pages:           input.PageBacklinks,
		TopicMap:        topicMap,
		Pagination:      NewPagination(1, input.PageBacklinkCount, int(PageBacklinkLimit)),
		SpaceIdentifier: input.SpaceIdentifier,
		PageNumber:      input.PageNumber,
	})

	return EditLinkData{
		LinkList:     linkListVM,
		BacklinkList: backlinkListVM,
	}
}
