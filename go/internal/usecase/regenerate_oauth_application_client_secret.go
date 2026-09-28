package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// RegenerateOAuthApplicationClientSecretUsecaseは、confidentialクライアントのシークレットを
// 発行し直す。
type RegenerateOAuthApplicationClientSecretUsecase struct {
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
}

// NewRegenerateOAuthApplicationClientSecretUsecaseはRegenerateOAuthApplicationClientSecretUsecaseを生成する
func NewRegenerateOAuthApplicationClientSecretUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
) *RegenerateOAuthApplicationClientSecretUsecase {
	return &RegenerateOAuthApplicationClientSecretUsecase{
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
		featureFlagRepo:      featureFlagRepo,
		oauthApplicationRepo: oauthApplicationRepo,
	}
}

// RegenerateOAuthApplicationClientSecretInputは再発行の入力パラメータ
type RegenerateOAuthApplicationClientSecretInput struct {
	SpaceIdentifier    model.SpaceIdentifier
	UserID             model.UserID
	OAuthApplicationID model.OAuthApplicationID
}

// RegenerateOAuthApplicationClientSecretOutputは再発行したシークレットを保持する
type RegenerateOAuthApplicationClientSecretOutput struct {
	Space       *model.Space
	Application *model.OAuthApplication

	// ClientSecretは新しいシークレットの値そのもの。データベースにはダイジェストしか残らない
	// ため、利用者に値を示せるのはこの再発行の応答だけである
	ClientSecret string
}

// Executeはconfidentialクライアントのシークレットを発行し直す。
//
// 以前のシークレットはダイジェストを置き換えた時点で照合できなくなる。シークレットが漏れた
// ときに再発行するため、以前のものを猶予期間を置いて残すことはしない。発行済みの許可と
// トークンはそのまま使える。編集の1つとしてoauth_application:writeで判定する。publicクライアント
// にはシークレットが無いため「見つからない」として答える。
func (uc *RegenerateOAuthApplicationClientSecretUsecase) Execute(ctx context.Context, input RegenerateOAuthApplicationClientSecretInput) (*RegenerateOAuthApplicationClientSecretOutput, error) {
	// 1. データ取得と認可チェック
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanUpdateOAuthApplication,
	)
	if err != nil {
		return nil, err
	}

	// 2. ビジネスロジック
	clientSecret, err := auth.GenerateOpaqueToken(auth.OAuthClientSecretPrefix)
	if err != nil {
		return nil, fmt.Errorf("クライアントシークレットの生成に失敗: %w", err)
	}

	// 3. 永続化。対象がconfidentialクライアントであることはRepositoryの条件で確かめる
	app, err := uc.oauthApplicationRepo.UpdateClientSecretDigest(ctx, input.OAuthApplicationID, access.space.ID, auth.DigestOpaqueToken(clientSecret))
	if err != nil {
		return nil, fmt.Errorf("クライアントシークレットの更新に失敗: %w", err)
	}
	if app == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	return &RegenerateOAuthApplicationClientSecretOutput{
		Space:        access.space,
		Application:  app,
		ClientSecret: clientSecret,
	}, nil
}
