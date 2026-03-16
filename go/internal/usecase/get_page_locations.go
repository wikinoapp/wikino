package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageLocationsUsecase はページロケーション検索のデータ取得ユースケース
type GetPageLocationsUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
}

// NewGetPageLocationsUsecase は GetPageLocationsUsecase を生成する
func NewGetPageLocationsUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
) *GetPageLocationsUsecase {
	return &GetPageLocationsUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
	}
}

// GetPageLocationsInput はページロケーション検索の入力パラメータ
type GetPageLocationsInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	Query           string
}

// GetPageLocationsOutput はページロケーション検索の出力
type GetPageLocationsOutput struct {
	Locations []repository.PageLocation
}

// Execute はページロケーションを検索する
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

	locations, err := uc.pageRepo.SearchPageLocations(ctx, space.ID, input.Query)
	if err != nil {
		return nil, fmt.Errorf("ページロケーションの検索に失敗: %w", err)
	}

	return &GetPageLocationsOutput{
		Locations: locations,
	}, nil
}
