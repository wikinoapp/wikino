package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// DeleteOAuthApplicationUsecaseは、スペースのOAuthアプリを削除する。
type DeleteOAuthApplicationUsecase struct {
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
}

// NewDeleteOAuthApplicationUsecaseはDeleteOAuthApplicationUsecaseを生成する
func NewDeleteOAuthApplicationUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
) *DeleteOAuthApplicationUsecase {
	return &DeleteOAuthApplicationUsecase{
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
		featureFlagRepo:      featureFlagRepo,
		oauthApplicationRepo: oauthApplicationRepo,
	}
}

// DeleteOAuthApplicationInputは削除の入力パラメータ
type DeleteOAuthApplicationInput struct {
	SpaceIdentifier    model.SpaceIdentifier
	UserID             model.UserID
	OAuthApplicationID model.OAuthApplicationID
}

// DeleteOAuthApplicationOutputは削除したアプリを保持する
type DeleteOAuthApplicationOutput struct {
	Space       *model.Space
	Application *model.OAuthApplication
}

// ExecuteはOAuthアプリを削除し、そのアプリの許可とトークンをすべて失効する。
//
// 作成したメンバーかどうかは問わない (oauth_application:deleteはスペースのアプリ全体を扱う
// 権限である)。行は消さずに削除日時を記録する。削除したアプリは一覧から消え、クライアントIDで
// 引けなくなるため、新しい認可も受け付けなくなる。別のスペースのアプリ・削除済みのアプリは
// 「見つからない」として答える。
func (uc *DeleteOAuthApplicationUsecase) Execute(ctx context.Context, input DeleteOAuthApplicationInput) (*DeleteOAuthApplicationOutput, error) {
	// 1. データ取得と認可チェック
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanDeleteOAuthApplication,
	)
	if err != nil {
		return nil, err
	}

	// 2. 永続化。アプリの削除と許可・トークンの失効はRepositoryが1文で行う
	app, err := uc.oauthApplicationRepo.Discard(ctx, input.OAuthApplicationID, access.space.ID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの削除に失敗: %w", err)
	}
	if app == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	return &DeleteOAuthApplicationOutput{
		Space:       access.space,
		Application: app,
	}, nil
}
