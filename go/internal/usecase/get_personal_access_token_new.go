package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPersonalAccessTokenNewUsecaseは、個人用アクセストークンの発行フォームが表示するものを集める。
type GetPersonalAccessTokenNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	featureFlagRepo *repository.FeatureFlagRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetPersonalAccessTokenNewUsecaseはGetPersonalAccessTokenNewUsecaseを生成する
func NewGetPersonalAccessTokenNewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetPersonalAccessTokenNewUsecase {
	return &GetPersonalAccessTokenNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		featureFlagRepo: featureFlagRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetPersonalAccessTokenNewInputは発行フォームの表示に必要な入力パラメータ
type GetPersonalAccessTokenNewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetPersonalAccessTokenNewOutputは発行フォームの表示に必要なデータを保持する
type GetPersonalAccessTokenNewOutput struct {
	Space *model.Space

	// IneffectiveScopesは、トークンに付与できるスコープのうちユーザーが今の権限では持たないもの。
	// 付与してもトークンでは効かないため、発行フォームで注記を添える
	IneffectiveScopes []model.Scope
}

// Executeはトークンを発行するスペースと、今の権限では効かないスコープを解決する
func (uc *GetPersonalAccessTokenNewUsecase) Execute(ctx context.Context, input GetPersonalAccessTokenNewInput) (*GetPersonalAccessTokenNewOutput, error) {
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanCreatePersonalAccessToken,
	)
	if err != nil {
		return nil, err
	}

	ineffective, err := ineffectiveAPITokenScopes(ctx, uc.topicMemberRepo, access.spaceMember, model.APITokenScopes)
	if err != nil {
		return nil, err
	}

	return &GetPersonalAccessTokenNewOutput{Space: access.space, IneffectiveScopes: ineffective}, nil
}
