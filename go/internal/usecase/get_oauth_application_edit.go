package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetOAuthApplicationEditUsecaseは、OAuthアプリの編集フォームが表示するものを集める。
type GetOAuthApplicationEditUsecase struct {
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
}

// NewGetOAuthApplicationEditUsecaseはGetOAuthApplicationEditUsecaseを生成する
func NewGetOAuthApplicationEditUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
) *GetOAuthApplicationEditUsecase {
	return &GetOAuthApplicationEditUsecase{
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
		featureFlagRepo:      featureFlagRepo,
		oauthApplicationRepo: oauthApplicationRepo,
	}
}

// GetOAuthApplicationEditInputは編集フォームの表示に必要な入力パラメータ
type GetOAuthApplicationEditInput struct {
	SpaceIdentifier    model.SpaceIdentifier
	UserID             model.UserID
	OAuthApplicationID model.OAuthApplicationID
}

// GetOAuthApplicationEditOutputは編集フォームの表示に必要なデータを保持する
type GetOAuthApplicationEditOutput struct {
	Space       *model.Space
	Application *model.OAuthApplication
}

// Executeはスペースと、編集するそのスペースのOAuthアプリを取得する。別のスペースのアプリ・
// 削除されたアプリ・公式クライアントは「見つからない」として答える
func (uc *GetOAuthApplicationEditUsecase) Execute(ctx context.Context, input GetOAuthApplicationEditInput) (*GetOAuthApplicationEditOutput, error) {
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanUpdateOAuthApplication,
	)
	if err != nil {
		return nil, err
	}

	app, err := uc.oauthApplicationRepo.FindByIDAndSpaceID(ctx, input.OAuthApplicationID, access.space.ID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの取得に失敗: %w", err)
	}
	if app == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	return &GetOAuthApplicationEditOutput{
		Space:       access.space,
		Application: app,
	}, nil
}
