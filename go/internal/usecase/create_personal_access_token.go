package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreatePersonalAccessTokenUsecaseは、スペースメンバーの個人アクセストークンを発行する。
type CreatePersonalAccessTokenUsecase struct {
	spaceRepo               *repository.SpaceRepository
	spaceMemberRepo         *repository.SpaceMemberRepository
	featureFlagRepo         *repository.FeatureFlagRepository
	personalAccessTokenRepo *repository.PersonalAccessTokenRepository
	createValidator         *validator.PersonalAccessTokenCreateValidator
}

// NewCreatePersonalAccessTokenUsecaseはCreatePersonalAccessTokenUsecaseを生成する
func NewCreatePersonalAccessTokenUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	personalAccessTokenRepo *repository.PersonalAccessTokenRepository,
	createValidator *validator.PersonalAccessTokenCreateValidator,
) *CreatePersonalAccessTokenUsecase {
	return &CreatePersonalAccessTokenUsecase{
		spaceRepo:               spaceRepo,
		spaceMemberRepo:         spaceMemberRepo,
		featureFlagRepo:         featureFlagRepo,
		personalAccessTokenRepo: personalAccessTokenRepo,
		createValidator:         createValidator,
	}
}

// CreatePersonalAccessTokenInputは発行の入力パラメータ。
// ScopesとExpirationDaysはフォームが送信した文字列で、変換はバリデーターが行う。
type CreatePersonalAccessTokenInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	Name            string
	Scopes          []string
	ExpirationDays  string
}

// CreatePersonalAccessTokenOutputは発行したトークンを保持する
type CreatePersonalAccessTokenOutput struct {
	Space *model.Space
	Token *model.PersonalAccessToken

	// TokenValueはトークンの値そのもの。データベースにはダイジェストしか残らないため、
	// 利用者に値を示せるのはこの発行の応答だけである
	TokenValue string
}

// Executeは個人アクセストークンを発行する
func (uc *CreatePersonalAccessTokenUsecase) Execute(ctx context.Context, input CreatePersonalAccessTokenInput) (*CreatePersonalAccessTokenOutput, error) {
	// 1. データ取得と認可チェック
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanCreatePersonalAccessToken,
	)
	if err != nil {
		return nil, err
	}

	// 2. バリデーション
	validated, err := uc.createValidator.Validate(ctx, validator.PersonalAccessTokenCreateValidatorInput{
		Name:           input.Name,
		Scopes:         input.Scopes,
		ExpirationDays: input.ExpirationDays,
	})
	if err != nil {
		return nil, err
	}

	// 3. 永続化
	tokenValue, err := auth.GenerateOpaqueToken(auth.PersonalAccessTokenPrefix)
	if err != nil {
		return nil, fmt.Errorf("トークンの生成に失敗: %w", err)
	}

	token, err := uc.personalAccessTokenRepo.Create(ctx, repository.CreatePersonalAccessTokenInput{
		SpaceID:        access.space.ID,
		SpaceMemberID:  access.spaceMember.ID,
		Name:           input.Name,
		TokenDigest:    auth.DigestOpaqueToken(tokenValue),
		TokenLastChars: auth.OpaqueTokenLastChars(tokenValue),
		Scopes:         validated.Scopes,
		ExpiresAt:      time.Now().AddDate(0, 0, validated.ExpirationDays),
	})
	if err != nil {
		return nil, fmt.Errorf("個人アクセストークンの作成に失敗: %w", err)
	}

	return &CreatePersonalAccessTokenOutput{
		Space:      access.space,
		Token:      token,
		TokenValue: tokenValue,
	}, nil
}
