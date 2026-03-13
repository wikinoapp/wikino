package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetLinkListUsecase はリンク一覧取得ユースケース
type GetLinkListUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	draftPageRepo   *repository.DraftPageRepository
}

// NewGetLinkListUsecase は GetLinkListUsecase を生成する
func NewGetLinkListUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
) *GetLinkListUsecase {
	return &GetLinkListUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		draftPageRepo:   draftPageRepo,
	}
}

// GetLinkListInput はリンク一覧取得の入力パラメータ
type GetLinkListInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
	CurrentPage     int32
	LinkLimit       int32
	BacklinkLimit   int32
}

// GetLinkListOutput はリンク一覧取得の出力
type GetLinkListOutput struct {
	Space            *model.Space
	SpaceMember      *model.SpaceMember
	Page             *model.Page
	TopicMember      *model.TopicMember
	LinkedPages      []*model.Page
	LinkedTotalCount int64
	BacklinksPerPage map[model.PageID]*EditLinkBacklinks
	TopicMap         map[model.TopicID]*model.Topic
}

// Execute はリンク一覧を取得する
func (uc *GetLinkListUsecase) Execute(ctx context.Context, input GetLinkListInput) (*GetLinkListOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, nil
	}

	pg, err := uc.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(input.PageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if pg == nil {
		return nil, nil
	}

	topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, pg.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	draftPage, err := uc.draftPageRepo.FindByPageAndMember(ctx, pg.ID, spaceMember.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
	}

	var linkedPageIDs []model.PageID
	if draftPage != nil {
		linkedPageIDs = draftPage.LinkedPageIDs
	} else {
		linkedPageIDs = pg.LinkedPageIDs
	}

	if len(linkedPageIDs) == 0 {
		return &GetLinkListOutput{
			Space:       space,
			SpaceMember: spaceMember,
			Page:        pg,
			TopicMember: topicMember,
		}, nil
	}

	paginatedLinks, err := uc.pageRepo.FindLinkedPagesPaginated(ctx, linkedPageIDs, space.ID, input.CurrentPage, input.LinkLimit)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの取得に失敗: %w", err)
	}

	excludePageIDs := buildExcludePageIDs(pg.ID, paginatedLinks.Pages)

	backlinkPaginatedMap, err := uc.pageRepo.FindBacklinksForPages(ctx, paginatedLinks.Pages, space.ID, input.BacklinkLimit, excludePageIDs)
	if err != nil {
		return nil, fmt.Errorf("バックリンクの取得に失敗: %w", err)
	}

	var allPageSlices [][]*model.Page
	allPageSlices = append(allPageSlices, paginatedLinks.Pages)
	for _, paginated := range backlinkPaginatedMap {
		allPageSlices = append(allPageSlices, paginated.Pages)
	}

	topicIDs := collectTopicIDsFromPages(allPageSlices...)
	topics, err := uc.topicRepo.FindByIDsAndSpace(ctx, topicIDs, space.ID)
	if err != nil {
		return nil, fmt.Errorf("トピックの一括取得に失敗: %w", err)
	}

	topicMap := make(map[model.TopicID]*model.Topic, len(topics))
	for _, t := range topics {
		topicMap[t.ID] = t
	}

	backlinksPerPage := make(map[model.PageID]*EditLinkBacklinks, len(backlinkPaginatedMap))
	for pageID, paginated := range backlinkPaginatedMap {
		backlinksPerPage[pageID] = &EditLinkBacklinks{
			Pages:      paginated.Pages,
			TotalCount: paginated.TotalCount,
		}
	}

	return &GetLinkListOutput{
		Space:            space,
		SpaceMember:      spaceMember,
		Page:             pg,
		TopicMember:      topicMember,
		LinkedPages:      paginatedLinks.Pages,
		LinkedTotalCount: paginatedLinks.TotalCount,
		BacklinksPerPage: backlinksPerPage,
		TopicMap:         topicMap,
	}, nil
}
