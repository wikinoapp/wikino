package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSpaceSettingsGeneralUsecaseはスペースの一般設定画面が表示するものを集める。
type GetSpaceSettingsGeneralUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
}

// NewGetSpaceSettingsGeneralUsecaseはGetSpaceSettingsGeneralUsecaseを生成する
func NewGetSpaceSettingsGeneralUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
) *GetSpaceSettingsGeneralUsecase {
	return &GetSpaceSettingsGeneralUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
	}
}

// GetSpaceSettingsGeneralInputは一般設定画面の表示に必要な入力パラメータ
type GetSpaceSettingsGeneralInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetSpaceSettingsGeneralOutputは一般設定画面の表示に必要なデータを保持する
type GetSpaceSettingsGeneralOutput struct {
	Space *model.Space
}

// Executeは一般設定を編集するスペースを解決する
func (uc *GetSpaceSettingsGeneralUsecase) Execute(ctx context.Context, input GetSpaceSettingsGeneralInput) (*GetSpaceSettingsGeneralOutput, error) {
	space, err := fetchSpaceUpdateAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	return &GetSpaceSettingsGeneralOutput{Space: space}, nil
}
