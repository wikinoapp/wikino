package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageBacklinksUsecase はページレベルのバックリンク一覧取得ユースケース
type GetPageBacklinksUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetPageBacklinksUsecase は GetPageBacklinksUsecase を生成する
func NewGetPageBacklinksUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetPageBacklinksUsecase {
	return &GetPageBacklinksUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetPageBacklinksInput はページレベルのバックリンク一覧取得の入力パラメータ
type GetPageBacklinksInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
	CurrentPage     int32
	Limit           int32
}

// GetPageBacklinksOutput はページレベルのバックリンク一覧取得の出力
type GetPageBacklinksOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	Page        *model.Page
	TopicMember *model.TopicMember
	Backlinks   []*model.Page
	TotalCount  int64
	TopicMap    map[model.TopicID]*model.Topic
}

// Execute はページレベルのバックリンク一覧を取得する
func (uc *GetPageBacklinksUsecase) Execute(ctx context.Context, input GetPageBacklinksInput) (*GetPageBacklinksOutput, error) {
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

	paginatedBacklinks, err := uc.pageRepo.FindBacklinkedPagesPaginated(ctx, pg.ID, space.ID, input.CurrentPage, input.Limit, nil)
	if err != nil {
		return nil, fmt.Errorf("ページレベルのバックリンクの取得に失敗: %w", err)
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

	return &GetPageBacklinksOutput{
		Space:       space,
		SpaceMember: spaceMember,
		Page:        pg,
		TopicMember: topicMember,
		Backlinks:   paginatedBacklinks.Pages,
		TotalCount:  paginatedBacklinks.TotalCount,
		TopicMap:    topicMap,
	}, nil
}
