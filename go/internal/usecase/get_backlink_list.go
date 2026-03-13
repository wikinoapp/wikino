package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetBacklinkListUsecase はバックリンク一覧取得ユースケース
type GetBacklinkListUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetBacklinkListUsecase は GetBacklinkListUsecase を生成する
func NewGetBacklinkListUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetBacklinkListUsecase {
	return &GetBacklinkListUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetBacklinkListInput はバックリンク一覧取得の入力パラメータ
type GetBacklinkListInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	PageNumber       int32
	LinkedPageNumber int32
	UserID           model.UserID
	CurrentPage      int32
	Limit            int32
}

// GetBacklinkListOutput はバックリンク一覧取得の出力
type GetBacklinkListOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	Page        *model.Page
	TopicMember *model.TopicMember
	LinkedPage  *model.Page
	Backlinks   []*model.Page
	TotalCount  int64
	TopicMap    map[model.TopicID]*model.Topic
}

// Execute はバックリンク一覧を取得する
func (uc *GetBacklinkListUsecase) Execute(ctx context.Context, input GetBacklinkListInput) (*GetBacklinkListOutput, error) {
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

	linkedPage, err := uc.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(input.LinkedPageNumber))
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの取得に失敗: %w", err)
	}
	if linkedPage == nil {
		return nil, nil
	}

	excludePageIDs := []model.PageID{pg.ID, linkedPage.ID}
	paginatedBacklinks, err := uc.pageRepo.FindBacklinkedPagesPaginated(ctx, linkedPage.ID, space.ID, input.CurrentPage, input.Limit, excludePageIDs)
	if err != nil {
		return nil, fmt.Errorf("バックリンクの取得に失敗: %w", err)
	}

	topicIDs := collectTopicIDsFromPages(paginatedBacklinks.Pages)
	topics, err := uc.topicRepo.FindByIDsAndSpace(ctx, topicIDs, space.ID)
	if err != nil {
		return nil, fmt.Errorf("トピックの一括取得に失敗: %w", err)
	}

	topicMap := make(map[model.TopicID]*model.Topic, len(topics))
	for _, t := range topics {
		topicMap[t.ID] = t
	}

	return &GetBacklinkListOutput{
		Space:       space,
		SpaceMember: spaceMember,
		Page:        pg,
		TopicMember: topicMember,
		LinkedPage:  linkedPage,
		Backlinks:   paginatedBacklinks.Pages,
		TotalCount:  paginatedBacklinks.TotalCount,
		TopicMap:    topicMap,
	}, nil
}
