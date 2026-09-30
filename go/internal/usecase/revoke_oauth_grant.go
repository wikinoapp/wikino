package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// RevokeOAuthGrantUsecaseは、スペースメンバー自身が許可した連携を解除する。
type RevokeOAuthGrantUsecase struct {
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthGrantRepo       *repository.OAuthGrantRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
}

// NewRevokeOAuthGrantUsecaseはRevokeOAuthGrantUsecaseを生成する
func NewRevokeOAuthGrantUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthGrantRepo *repository.OAuthGrantRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
) *RevokeOAuthGrantUsecase {
	return &RevokeOAuthGrantUsecase{
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
		featureFlagRepo:      featureFlagRepo,
		oauthGrantRepo:       oauthGrantRepo,
		oauthApplicationRepo: oauthApplicationRepo,
	}
}

// RevokeOAuthGrantInputは連携の解除の入力パラメータ
type RevokeOAuthGrantInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	OAuthGrantID    model.OAuthGrantID
}

// RevokeOAuthGrantOutputは解除した許可と、そのアプリを保持する
type RevokeOAuthGrantOutput struct {
	Space *model.Space
	Grant *model.OAuthGrant

	// Applicationは許可したアプリ。解除の後に削除されたなどで見つからなければnilになる
	Application *model.OAuthApplication
}

// Executeは許可と、許可に属するアクセストークン・リフレッシュトークンをすべて失効する。
//
// 解除できるのは閲覧者自身の許可だけである。他のメンバーの許可や、既に失効した許可は
// 「見つからない」として答える。解除した許可は一覧から消えるため、同じ許可への2度目の
// 解除は存在しないものへの操作と区別できない。
func (uc *RevokeOAuthGrantUsecase) Execute(ctx context.Context, input RevokeOAuthGrantInput) (*RevokeOAuthGrantOutput, error) {
	// 1. データ取得と認可チェック
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanDeleteOAuthGrant,
	)
	if err != nil {
		return nil, err
	}

	// 2. 永続化
	grant, err := uc.oauthGrantRepo.Revoke(ctx, input.OAuthGrantID, access.space.ID, access.spaceMember.ID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可の失効に失敗: %w", err)
	}
	if grant == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	// 3. 完了の表示に使うアプリ名を引く
	apps, err := uc.oauthApplicationRepo.ListAvailableInSpaceByIDs(ctx, []model.OAuthApplicationID{grant.OAuthApplicationID}, access.space.ID)
	if err != nil {
		return nil, fmt.Errorf("許可したOAuthアプリの取得に失敗: %w", err)
	}
	var app *model.OAuthApplication
	if len(apps) > 0 {
		app = apps[0]
	}

	return &RevokeOAuthGrantOutput{
		Space:       access.space,
		Grant:       grant,
		Application: app,
	}, nil
}
