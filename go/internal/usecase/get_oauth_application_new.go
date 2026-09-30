package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetOAuthApplicationNewUsecaseは、OAuthアプリの登録フォームが表示するものを集める。
type GetOAuthApplicationNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	featureFlagRepo *repository.FeatureFlagRepository
}

// NewGetOAuthApplicationNewUsecaseはGetOAuthApplicationNewUsecaseを生成する
func NewGetOAuthApplicationNewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
) *GetOAuthApplicationNewUsecase {
	return &GetOAuthApplicationNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		featureFlagRepo: featureFlagRepo,
	}
}

// GetOAuthApplicationNewInputは登録フォームの表示に必要な入力パラメータ
type GetOAuthApplicationNewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetOAuthApplicationNewOutputは登録フォームの表示に必要なデータを保持する
type GetOAuthApplicationNewOutput struct {
	Space *model.Space
}

// Executeはアプリを登録するスペースを解決する
func (uc *GetOAuthApplicationNewUsecase) Execute(ctx context.Context, input GetOAuthApplicationNewInput) (*GetOAuthApplicationNewOutput, error) {
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanCreateOAuthApplication,
	)
	if err != nil {
		return nil, err
	}

	return &GetOAuthApplicationNewOutput{Space: access.space}, nil
}
