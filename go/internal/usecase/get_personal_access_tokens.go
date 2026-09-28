package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPersonalAccessTokensUsecaseは、個人アクセストークンの一覧の画面が表示するものを集める。
type GetPersonalAccessTokensUsecase struct {
	spaceRepo               *repository.SpaceRepository
	spaceMemberRepo         *repository.SpaceMemberRepository
	featureFlagRepo         *repository.FeatureFlagRepository
	personalAccessTokenRepo *repository.PersonalAccessTokenRepository
}

// NewGetPersonalAccessTokensUsecaseはGetPersonalAccessTokensUsecaseを生成する
func NewGetPersonalAccessTokensUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	personalAccessTokenRepo *repository.PersonalAccessTokenRepository,
) *GetPersonalAccessTokensUsecase {
	return &GetPersonalAccessTokensUsecase{
		spaceRepo:               spaceRepo,
		spaceMemberRepo:         spaceMemberRepo,
		featureFlagRepo:         featureFlagRepo,
		personalAccessTokenRepo: personalAccessTokenRepo,
	}
}

// GetPersonalAccessTokensInputは一覧の画面を描画するための入力パラメータ
type GetPersonalAccessTokensInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetPersonalAccessTokensOutputは一覧の画面に表示するデータを保持する
type GetPersonalAccessTokensOutput struct {
	Space *model.Space

	// Tokensは閲覧者自身の失効していないトークンで、新しい順に並ぶ。期限切れのものも含む
	Tokens []*model.PersonalAccessToken

	// CanCreateは閲覧者が新しいトークンを発行できるか (personal_access_token:write) を表す
	CanCreate bool

	// CanDeleteは閲覧者が自分のトークンを失効できるか (personal_access_token:delete) を表す
	CanDelete bool
}

// Executeはスペースと、閲覧者自身の個人アクセストークンの一覧を取得する。
//
// 一覧に出すのは閲覧者自身のトークンだけである。他のメンバーのトークンを管理者が一覧する
// 機能は、メンバーを招待できるようになってから別のユースケースで扱う。
func (uc *GetPersonalAccessTokensUsecase) Execute(ctx context.Context, input GetPersonalAccessTokensInput) (*GetPersonalAccessTokensOutput, error) {
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanShowPersonalAccessTokens,
	)
	if err != nil {
		return nil, err
	}

	tokens, err := uc.personalAccessTokenRepo.ListUnrevokedBySpaceMember(ctx, access.space.ID, access.spaceMember.ID)
	if err != nil {
		return nil, fmt.Errorf("個人アクセストークンの一覧の取得に失敗: %w", err)
	}

	return &GetPersonalAccessTokensOutput{
		Space:     access.space,
		Tokens:    tokens,
		CanCreate: access.authorizer.CanCreatePersonalAccessToken(),
		CanDelete: access.authorizer.CanDeletePersonalAccessToken(),
	}, nil
}
