package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetEditLinkDataUsecase は編集画面のリンクデータ取得ユースケース
type GetEditLinkDataUsecase struct {
	pageRepo  *repository.PageRepository
	topicRepo *repository.TopicRepository
}

// NewGetEditLinkDataUsecase は GetEditLinkDataUsecase を生成する
func NewGetEditLinkDataUsecase(
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
) *GetEditLinkDataUsecase {
	return &GetEditLinkDataUsecase{
		pageRepo:  pageRepo,
		topicRepo: topicRepo,
	}
}

// GetEditLinkDataInput は編集画面のリンクデータ取得の入力パラメータ
type GetEditLinkDataInput struct {
	Page              *model.Page
	DraftPage         *model.DraftPage
	SpaceID           model.SpaceID
	CurrentPage       int32
	LinkLimit         int32
	BacklinkLimit     int32
	PageBacklinkLimit int32
}

// EditLinkBacklinks はバックリンクのページスライスと総件数のペア
type EditLinkBacklinks struct {
	Pages      []*model.Page
	TotalCount int64
}

// GetEditLinkDataOutput は編集画面のリンクデータ取得の出力
type GetEditLinkDataOutput struct {
	LinkedPages       []*model.Page
	LinkedTotalCount  int64
	BacklinksPerPage  map[model.PageID]*EditLinkBacklinks
	PageBacklinks     []*model.Page
	PageBacklinkCount int64
	LinkTopics        []*model.Topic
}

// Execute は編集画面のリンク・バックリンクデータを取得する
func (uc *GetEditLinkDataUsecase) Execute(ctx context.Context, input GetEditLinkDataInput) (*GetEditLinkDataOutput, error) {
	var linkedPageIDs []model.PageID
	if input.DraftPage != nil {
		linkedPageIDs = input.DraftPage.LinkedPageIDs
	} else {
		linkedPageIDs = input.Page.LinkedPageIDs
	}

	var paginatedLinks *repository.PaginatedPages
	var backlinkPaginatedMap map[model.PageID]*repository.PaginatedPages
	if len(linkedPageIDs) > 0 {
		var err error
		paginatedLinks, err = uc.pageRepo.FindLinkedPagesPaginated(ctx, linkedPageIDs, input.SpaceID, input.CurrentPage, input.LinkLimit)
		if err != nil {
			return nil, fmt.Errorf("リンク先ページの取得に失敗: %w", err)
		}

		excludePageIDs := buildExcludePageIDs(input.Page.ID, paginatedLinks.Pages)

		backlinkPaginatedMap, err = uc.pageRepo.FindBacklinksForPages(ctx, paginatedLinks.Pages, input.SpaceID, input.BacklinkLimit, excludePageIDs)
		if err != nil {
			return nil, fmt.Errorf("バックリンクの取得に失敗: %w", err)
		}
	}

	paginatedBacklinks, err := uc.pageRepo.FindBacklinkedPagesPaginated(ctx, input.Page.ID, input.SpaceID, 1, input.PageBacklinkLimit, nil)
	if err != nil {
		return nil, fmt.Errorf("ページレベルのバックリンクの取得に失敗: %w", err)
	}

	// すべてのページのTopicIDを収集してトピックを一括取得
	var allPageSlices [][]*model.Page
	if paginatedLinks != nil {
		allPageSlices = append(allPageSlices, paginatedLinks.Pages)
	}
	for _, paginated := range backlinkPaginatedMap {
		allPageSlices = append(allPageSlices, paginated.Pages)
	}
	allPageSlices = append(allPageSlices, paginatedBacklinks.Pages)

	topicIDs := collectTopicIDsFromPages(allPageSlices...)
	topics, err := uc.topicRepo.FindByIDsAndSpace(ctx, topicIDs, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("トピックの一括取得に失敗: %w", err)
	}

	var linkedPages []*model.Page
	var linkedTotalCount int64
	if paginatedLinks != nil {
		linkedPages = paginatedLinks.Pages
		linkedTotalCount = paginatedLinks.TotalCount
	}

	backlinksPerPage := make(map[model.PageID]*EditLinkBacklinks, len(backlinkPaginatedMap))
	for pageID, paginated := range backlinkPaginatedMap {
		backlinksPerPage[pageID] = &EditLinkBacklinks{
			Pages:      paginated.Pages,
			TotalCount: paginated.TotalCount,
		}
	}

	return &GetEditLinkDataOutput{
		LinkedPages:       linkedPages,
		LinkedTotalCount:  linkedTotalCount,
		BacklinksPerPage:  backlinksPerPage,
		PageBacklinks:     paginatedBacklinks.Pages,
		PageBacklinkCount: paginatedBacklinks.TotalCount,
		LinkTopics:        topics,
	}, nil
}

// buildExcludePageIDs は編集中のページ自身とリンク先ページからバックリンク除外用のPageIDスライスを構築する
func buildExcludePageIDs(currentPageID model.PageID, linkedPages []*model.Page) []model.PageID {
	ids := make([]model.PageID, 0, 1+len(linkedPages))
	ids = append(ids, currentPageID)
	for _, p := range linkedPages {
		ids = append(ids, p.ID)
	}
	return ids
}

// collectTopicIDsFromPages は複数のページスライスからユニークなTopicIDを収集する
func collectTopicIDsFromPages(pageSlices ...[]*model.Page) []model.TopicID {
	topicIDSet := make(map[model.TopicID]struct{})
	for _, pages := range pageSlices {
		for _, p := range pages {
			topicIDSet[p.TopicID] = struct{}{}
		}
	}
	topicIDs := make([]model.TopicID, 0, len(topicIDSet))
	for id := range topicIDSet {
		topicIDs = append(topicIDs, id)
	}
	return topicIDs
}
