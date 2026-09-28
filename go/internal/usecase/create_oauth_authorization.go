package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateOAuthAuthorizationUsecaseは、同意画面での利用者の決定を受け、許可なら認可コードを発行する。
type CreateOAuthAuthorizationUsecase struct {
	cfg                        *config.Config
	featureFlagRepo            *repository.FeatureFlagRepository
	oauthApplicationRepo       *repository.OAuthApplicationRepository
	spaceRepo                  *repository.SpaceRepository
	spaceMemberRepo            *repository.SpaceMemberRepository
	oauthAuthorizationCodeRepo *repository.OAuthAuthorizationCodeRepository
}

// NewCreateOAuthAuthorizationUsecaseはCreateOAuthAuthorizationUsecaseを生成する
func NewCreateOAuthAuthorizationUsecase(
	cfg *config.Config,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	oauthAuthorizationCodeRepo *repository.OAuthAuthorizationCodeRepository,
) *CreateOAuthAuthorizationUsecase {
	return &CreateOAuthAuthorizationUsecase{
		cfg:                        cfg,
		featureFlagRepo:            featureFlagRepo,
		oauthApplicationRepo:       oauthApplicationRepo,
		spaceRepo:                  spaceRepo,
		spaceMemberRepo:            spaceMemberRepo,
		oauthAuthorizationCodeRepo: oauthAuthorizationCodeRepo,
	}
}

// CreateOAuthAuthorizationInputは同意の送信の入力パラメータ。Paramsは同意画面が引き継いだ
// 認可要求のパラメーターで、表示のときと同じ検証をやり直す
type CreateOAuthAuthorizationInput struct {
	UserID   model.UserID
	Params   OAuthAuthorizationParams
	Approved bool
}

// CreateOAuthAuthorizationOutputは、認可コードを付けてクライアントへ戻すための値
type CreateOAuthAuthorizationOutput struct {
	RedirectURI string
	State       string

	// Codeは認可コードの値そのもの。データベースにはダイジェストしか残らない
	Code string
}

// Executeは、許可なら認可コードを発行して返す。拒否した場合と、送信の時点でユーザーが
// 連携先のスペースでアプリを許可できない場合は、access_deniedの*model.OAuthAuthorizationErrorを
// 返す。要求が不正な場合とフィーチャーフラグが無効な場合はGetOAuthAuthorizationNewUsecaseと同じ
func (uc *CreateOAuthAuthorizationUsecase) Execute(ctx context.Context, input CreateOAuthAuthorizationInput) (*CreateOAuthAuthorizationOutput, error) {
	// 1. データ取得と検証
	if err := checkPublicAPIEnabled(ctx, uc.featureFlagRepo, input.UserID); err != nil {
		return nil, err
	}

	req, err := resolveOAuthAuthorizationRequest(ctx, uc.cfg.AppURL(), uc.oauthApplicationRepo, uc.spaceRepo, input.Params)
	if err != nil {
		return nil, err
	}

	accessDenied := &model.OAuthAuthorizationError{
		Code:        model.OAuthAuthorizationErrorAccessDenied,
		RedirectURI: req.redirectURI,
		State:       req.state,
	}
	if !input.Approved {
		return nil, accessDenied
	}

	// 2. 認可チェック。同意画面を出した後にメンバーから外されたり権限を失ったりしていないかを、
	// 送信の時点で確かめ直す
	decision, err := decideOAuthAuthorization(ctx, uc.spaceMemberRepo, req.space, input.UserID)
	if err != nil {
		return nil, err
	}
	if decision.denied != "" {
		return nil, accessDenied
	}

	// 3. 永続化
	code, err := auth.GenerateOpaqueToken(auth.OAuthAuthorizationCodePrefix)
	if err != nil {
		return nil, fmt.Errorf("認可コードの生成に失敗: %w", err)
	}

	_, err = uc.oauthAuthorizationCodeRepo.CreateWithGrant(ctx, repository.CreateOAuthAuthorizationCodeWithGrantInput{
		OAuthApplicationID: req.application.ID,
		SpaceID:            req.space.ID,
		SpaceMemberID:      decision.spaceMember.ID,
		CodeDigest:         auth.DigestOpaqueToken(code),
		Scopes:             req.scopes,
		RedirectURI:        req.redirectURI,
		CodeChallenge:      req.codeChallenge,
		ExpiresAt:          time.Now().Add(model.OAuthAuthorizationCodeLifetime),
	})
	if err != nil {
		return nil, fmt.Errorf("認可コードの作成に失敗: %w", err)
	}

	return &CreateOAuthAuthorizationOutput{
		RedirectURI: req.redirectURI,
		State:       req.state,
		Code:        code,
	}, nil
}
