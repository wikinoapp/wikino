package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateOAuthApplicationUsecaseは、スペースのOAuthアプリを登録する。
type CreateOAuthApplicationUsecase struct {
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
	createValidator      *validator.OAuthApplicationCreateValidator
}

// NewCreateOAuthApplicationUsecaseはCreateOAuthApplicationUsecaseを生成する
func NewCreateOAuthApplicationUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	createValidator *validator.OAuthApplicationCreateValidator,
) *CreateOAuthApplicationUsecase {
	return &CreateOAuthApplicationUsecase{
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
		featureFlagRepo:      featureFlagRepo,
		oauthApplicationRepo: oauthApplicationRepo,
		createValidator:      createValidator,
	}
}

// CreateOAuthApplicationInputは登録の入力パラメータ。
// RedirectURIsとClientTypeはフォームが送信した文字列で、変換はバリデーターが行う。
type CreateOAuthApplicationInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	Name            string
	RedirectURIs    string
	ClientType      string
}

// CreateOAuthApplicationOutputは登録したアプリを保持する
type CreateOAuthApplicationOutput struct {
	Space       *model.Space
	Application *model.OAuthApplication

	// ClientSecretはconfidentialクライアントのシークレットの値そのもので、publicクライアント
	// では空文字列になる。データベースにはダイジェストしか残らないため、利用者に値を示せるのは
	// この登録の応答だけである
	ClientSecret string
}

// ExecuteはOAuthアプリを登録する。作成したメンバーとして閲覧者のメンバーを記録する
func (uc *CreateOAuthApplicationUsecase) Execute(ctx context.Context, input CreateOAuthApplicationInput) (*CreateOAuthApplicationOutput, error) {
	// 1. データ取得と認可チェック
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanCreateOAuthApplication,
	)
	if err != nil {
		return nil, err
	}

	// 2. バリデーション
	validated, err := uc.createValidator.Validate(ctx, validator.OAuthApplicationCreateValidatorInput{
		Name:         input.Name,
		RedirectURIs: input.RedirectURIs,
		ClientType:   input.ClientType,
	})
	if err != nil {
		return nil, err
	}

	// 3. 永続化
	clientID, err := auth.GenerateOAuthClientID()
	if err != nil {
		return nil, fmt.Errorf("クライアントIDの生成に失敗: %w", err)
	}

	var clientSecret string
	var clientSecretDigest *string
	if validated.ClientType == model.OAuthClientTypeConfidential {
		clientSecret, err = auth.GenerateOpaqueToken(auth.OAuthClientSecretPrefix)
		if err != nil {
			return nil, fmt.Errorf("クライアントシークレットの生成に失敗: %w", err)
		}
		digest := auth.DigestOpaqueToken(clientSecret)
		clientSecretDigest = &digest
	}

	app, err := uc.oauthApplicationRepo.Create(ctx, repository.CreateOAuthApplicationInput{
		SpaceID:              access.space.ID,
		CreatedSpaceMemberID: access.spaceMember.ID,
		Name:                 input.Name,
		ClientID:             clientID,
		ClientSecretDigest:   clientSecretDigest,
		ClientType:           validated.ClientType,
		RedirectURIs:         validated.RedirectURIs,
	})
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの作成に失敗: %w", err)
	}

	return &CreateOAuthApplicationOutput{
		Space:        access.space,
		Application:  app,
		ClientSecret: clientSecret,
	}, nil
}
