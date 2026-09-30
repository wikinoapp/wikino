package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// RevokePersonalAccessTokenUsecaseは、スペースメンバー自身の個人アクセストークンを失効する。
type RevokePersonalAccessTokenUsecase struct {
	spaceRepo               *repository.SpaceRepository
	spaceMemberRepo         *repository.SpaceMemberRepository
	featureFlagRepo         *repository.FeatureFlagRepository
	personalAccessTokenRepo *repository.PersonalAccessTokenRepository
}

// NewRevokePersonalAccessTokenUsecaseはRevokePersonalAccessTokenUsecaseを生成する
func NewRevokePersonalAccessTokenUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	personalAccessTokenRepo *repository.PersonalAccessTokenRepository,
) *RevokePersonalAccessTokenUsecase {
	return &RevokePersonalAccessTokenUsecase{
		spaceRepo:               spaceRepo,
		spaceMemberRepo:         spaceMemberRepo,
		featureFlagRepo:         featureFlagRepo,
		personalAccessTokenRepo: personalAccessTokenRepo,
	}
}

// RevokePersonalAccessTokenInputは失効の入力パラメータ
type RevokePersonalAccessTokenInput struct {
	SpaceIdentifier       model.SpaceIdentifier
	UserID                model.UserID
	PersonalAccessTokenID model.PersonalAccessTokenID
}

// RevokePersonalAccessTokenOutputは失効したトークンを保持する
type RevokePersonalAccessTokenOutput struct {
	Space *model.Space
	Token *model.PersonalAccessToken
}

// Executeは個人アクセストークンを失効する。
//
// 失効できるのは閲覧者自身のトークンだけである。他のメンバーのトークンや、既に失効した
// トークンは「見つからない」として答える。失効したトークンは一覧から消えるため、同じ
// トークンへの2度目の失効は存在しないものへの操作と区別できない。
func (uc *RevokePersonalAccessTokenUsecase) Execute(ctx context.Context, input RevokePersonalAccessTokenInput) (*RevokePersonalAccessTokenOutput, error) {
	// 1. データ取得と認可チェック
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanDeletePersonalAccessToken,
	)
	if err != nil {
		return nil, err
	}

	// 2. 永続化
	token, err := uc.personalAccessTokenRepo.Revoke(ctx, input.PersonalAccessTokenID, access.space.ID, access.spaceMember.ID)
	if err != nil {
		return nil, fmt.Errorf("個人アクセストークンの失効に失敗: %w", err)
	}
	if token == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	return &RevokePersonalAccessTokenOutput{
		Space: access.space,
		Token: token,
	}, nil
}
