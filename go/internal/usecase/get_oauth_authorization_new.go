package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetOAuthAuthorizationNewUsecaseは、OAuthの認可要求を検証し、同意画面に示す内容を取得する。
type GetOAuthAuthorizationNewUsecase struct {
	cfg                  *config.Config
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
}

// NewGetOAuthAuthorizationNewUsecaseはGetOAuthAuthorizationNewUsecaseを生成する
func NewGetOAuthAuthorizationNewUsecase(
	cfg *config.Config,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
) *GetOAuthAuthorizationNewUsecase {
	return &GetOAuthAuthorizationNewUsecase{
		cfg:                  cfg,
		featureFlagRepo:      featureFlagRepo,
		oauthApplicationRepo: oauthApplicationRepo,
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
	}
}

// GetOAuthAuthorizationNewInputは同意画面の入力パラメータ
type GetOAuthAuthorizationNewInput struct {
	UserID model.UserID
	Params OAuthAuthorizationParams
}

// GetOAuthAuthorizationNewOutputは同意画面に示す内容
type GetOAuthAuthorizationNewOutput struct {
	Application *model.OAuthApplication
	Space       *model.Space
	Scopes      []model.Scope
	RedirectURI string

	// DeniedReasonは、ログイン中のユーザーが連携先のスペースでアプリを許可できない理由。
	// 空でなければ、同意画面の代わりに理由を示す
	DeniedReason OAuthAuthorizationDeniedReason
}

// Executeは認可要求を検証し、同意画面に示す内容を返す。要求が不正なら
// *model.OAuthAuthorizationErrorを、フィーチャーフラグが無効なら*model.AppErrorを返す
func (uc *GetOAuthAuthorizationNewUsecase) Execute(ctx context.Context, input GetOAuthAuthorizationNewInput) (*GetOAuthAuthorizationNewOutput, error) {
	// 1. データ取得と検証
	if err := checkPublicAPIEnabled(ctx, uc.featureFlagRepo, input.UserID); err != nil {
		return nil, err
	}

	req, err := resolveOAuthAuthorizationRequest(ctx, uc.cfg.AppURL(), uc.oauthApplicationRepo, uc.spaceRepo, input.Params)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック (許可できない場合も、理由を示してクライアントへ戻れるよう結果として返す)
	decision, err := decideOAuthAuthorization(ctx, uc.spaceMemberRepo, req.space, input.UserID)
	if err != nil {
		return nil, err
	}

	return &GetOAuthAuthorizationNewOutput{
		Application:  req.application,
		Space:        req.space,
		Scopes:       req.scopes,
		RedirectURI:  req.redirectURI,
		DeniedReason: decision.denied,
	}, nil
}
