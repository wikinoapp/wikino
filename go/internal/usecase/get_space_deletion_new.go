package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSpaceDeletionNewUsecaseはスペースの削除の確認画面が表示するものを集める。
type GetSpaceDeletionNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
}

// NewGetSpaceDeletionNewUsecaseはGetSpaceDeletionNewUsecaseを生成する。
func NewGetSpaceDeletionNewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
) *GetSpaceDeletionNewUsecase {
	return &GetSpaceDeletionNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
	}
}

// GetSpaceDeletionNewInputは確認画面を描画するための入力パラメータ。
type GetSpaceDeletionNewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetSpaceDeletionNewOutputは削除するスペースを返す。
type GetSpaceDeletionNewOutput struct {
	Space *model.Space
}

// Executeは削除するスペースを解決する。スペースを削除できない閲覧者は拒否する。
func (uc *GetSpaceDeletionNewUsecase) Execute(ctx context.Context, input GetSpaceDeletionNewInput) (*GetSpaceDeletionNewOutput, error) {
	space, err := fetchSpaceDeleteAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	return &GetSpaceDeletionNewOutput{Space: space}, nil
}
