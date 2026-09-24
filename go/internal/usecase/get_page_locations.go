package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageLocationsUsecaseはページロケーション検索のデータ取得ユースケース
type GetPageLocationsUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetPageLocationsUsecaseはGetPageLocationsUsecaseを生成する
func NewGetPageLocationsUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetPageLocationsUsecase {
	return &GetPageLocationsUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetPageLocationsInputはページロケーション検索の入力パラメータ
type GetPageLocationsInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	Query           string
}

// GetPageLocationsOutputはページロケーション検索の出力
type GetPageLocationsOutput struct {
	Locations []repository.PageLocation
}

// Executeはページロケーションを検索する
func (uc *GetPageLocationsUsecase) Execute(ctx context.Context, input GetPageLocationsInput) (*GetPageLocationsOutput, error) {
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

	// 開けないトピックのページタイトルを、キーワードを打つだけで探せないようにする
	access, err := fetchTopicAccess(ctx, uc.pageAccessRepos(), space.ID, spaceMember)
	if err != nil {
		return nil, err
	}

	locations, err := uc.pageRepo.SearchPageLocations(ctx, space.ID, spaceMember.ID, access.visibility(), input.Query)
	if err != nil {
		return nil, fmt.Errorf("ページロケーションの検索に失敗: %w", err)
	}

	return &GetPageLocationsOutput{
		Locations: locations,
	}, nil
}

func (uc *GetPageLocationsUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}
