package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetOAuthApplicationUsecaseは、OAuthアプリの詳細の画面が表示するものを集める。
type GetOAuthApplicationUsecase struct {
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
	userRepo             *repository.UserRepository
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
}

// NewGetOAuthApplicationUsecaseはGetOAuthApplicationUsecaseを生成する
func NewGetOAuthApplicationUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	userRepo *repository.UserRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
) *GetOAuthApplicationUsecase {
	return &GetOAuthApplicationUsecase{
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
		userRepo:             userRepo,
		featureFlagRepo:      featureFlagRepo,
		oauthApplicationRepo: oauthApplicationRepo,
	}
}

// GetOAuthApplicationInputは詳細の画面を描画するための入力パラメータ
type GetOAuthApplicationInput struct {
	SpaceIdentifier    model.SpaceIdentifier
	UserID             model.UserID
	OAuthApplicationID model.OAuthApplicationID
}

// GetOAuthApplicationOutputは詳細の画面に表示するデータを保持する
type GetOAuthApplicationOutput struct {
	Space       *model.Space
	Application *model.OAuthApplication

	// Creatorsはアプリを作成したメンバーのユーザー。作成したメンバーがいなくなっていれば空になる
	Creators map[model.SpaceMemberID]*model.User

	// CanUpdateは閲覧者がアプリを編集し、シークレットを再発行できるか (oauth_application:write) を表す
	CanUpdate bool

	// CanDeleteは閲覧者がアプリを削除できるか (oauth_application:delete) を表す
	CanDelete bool
}

// Executeはスペースと、そのスペースのOAuthアプリ1件を取得する。別のスペースのアプリ・
// 削除されたアプリ・公式クライアントは「見つからない」として答える
func (uc *GetOAuthApplicationUsecase) Execute(ctx context.Context, input GetOAuthApplicationInput) (*GetOAuthApplicationOutput, error) {
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanShowOAuthApplications,
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

	creators, err := buildOAuthApplicationCreators(ctx, uc.spaceMemberRepo, uc.userRepo, []*model.OAuthApplication{app}, access.space.ID)
	if err != nil {
		return nil, err
	}

	return &GetOAuthApplicationOutput{
		Space:       access.space,
		Application: app,
		Creators:    creators,
		CanUpdate:   access.authorizer.CanUpdateOAuthApplication(),
		CanDelete:   access.authorizer.CanDeleteOAuthApplication(),
	}, nil
}
