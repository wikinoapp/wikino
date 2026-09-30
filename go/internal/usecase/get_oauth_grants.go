package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetOAuthGrantsUsecaseは、連携中のアプリの一覧の画面が表示するものを集める。
type GetOAuthGrantsUsecase struct {
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthGrantRepo       *repository.OAuthGrantRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
}

// NewGetOAuthGrantsUsecaseはGetOAuthGrantsUsecaseを生成する
func NewGetOAuthGrantsUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthGrantRepo *repository.OAuthGrantRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
) *GetOAuthGrantsUsecase {
	return &GetOAuthGrantsUsecase{
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
		featureFlagRepo:      featureFlagRepo,
		oauthGrantRepo:       oauthGrantRepo,
		oauthApplicationRepo: oauthApplicationRepo,
	}
}

// GetOAuthGrantsInputは一覧の画面を描画するための入力パラメータ
type GetOAuthGrantsInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetOAuthGrantsOutputは一覧の画面に表示するデータを保持する
type GetOAuthGrantsOutput struct {
	Space *model.Space

	// Grantsは閲覧者自身の失効していない許可で、新しい順に並ぶ
	Grants []*model.OAuthGrant

	// Applicationsは許可したアプリ。削除されたアプリは含まない (削除したアプリの許可は
	// 削除のときに失効しているため、Grantsにも出ない)
	Applications map[model.OAuthApplicationID]*model.OAuthApplication

	// CanDeleteは閲覧者が連携を解除できるか (oauth_grant:delete) を表す
	CanDelete bool
}

// Executeはスペースと、閲覧者自身が許可した連携中のアプリの一覧を取得する。
//
// 一覧に出すのは閲覧者自身の許可だけである。他のメンバーの連携を管理者が一覧する機能は、
// メンバーを招待できるようになってから別のユースケースで扱う。
func (uc *GetOAuthGrantsUsecase) Execute(ctx context.Context, input GetOAuthGrantsInput) (*GetOAuthGrantsOutput, error) {
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanShowOAuthGrants,
	)
	if err != nil {
		return nil, err
	}

	grants, err := uc.oauthGrantRepo.ListUnrevokedBySpaceMember(ctx, access.space.ID, access.spaceMember.ID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可の一覧の取得に失敗: %w", err)
	}

	appIDs := make([]model.OAuthApplicationID, len(grants))
	for i, grant := range grants {
		appIDs[i] = grant.OAuthApplicationID
	}
	apps, err := uc.oauthApplicationRepo.ListAvailableInSpaceByIDs(ctx, appIDs, access.space.ID)
	if err != nil {
		return nil, fmt.Errorf("許可したOAuthアプリの取得に失敗: %w", err)
	}
	appMap := make(map[model.OAuthApplicationID]*model.OAuthApplication, len(apps))
	for _, app := range apps {
		appMap[app.ID] = app
	}

	return &GetOAuthGrantsOutput{
		Space:        access.space,
		Grants:       grants,
		Applications: appMap,
		CanDelete:    access.authorizer.CanDeleteOAuthGrant(),
	}, nil
}
