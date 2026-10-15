package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateOAuthAuthorizationUsecaseは、同意画面での利用者の決定を受け、許可なら認可コードを発行する。
type CreateOAuthAuthorizationUsecase struct {
	db                         *sql.DB
	cfg                        *config.Config
	oauthGrantRepo             *repository.OAuthGrantRepository
	featureFlagRepo            *repository.FeatureFlagRepository
	oauthApplicationRepo       *repository.OAuthApplicationRepository
	spaceRepo                  *repository.SpaceRepository
	spaceMemberRepo            *repository.SpaceMemberRepository
	oauthAuthorizationCodeRepo *repository.OAuthAuthorizationCodeRepository
	createValidator            *validator.OAuthAuthorizationCreateValidator
}

// NewCreateOAuthAuthorizationUsecaseはCreateOAuthAuthorizationUsecaseを生成する
func NewCreateOAuthAuthorizationUsecase(
	db *sql.DB,
	oauthGrantRepo *repository.OAuthGrantRepository,
	cfg *config.Config,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	oauthAuthorizationCodeRepo *repository.OAuthAuthorizationCodeRepository,
	createValidator *validator.OAuthAuthorizationCreateValidator,
) *CreateOAuthAuthorizationUsecase {
	return &CreateOAuthAuthorizationUsecase{
		db:                         db,
		oauthGrantRepo:             oauthGrantRepo,
		cfg:                        cfg,
		featureFlagRepo:            featureFlagRepo,
		oauthApplicationRepo:       oauthApplicationRepo,
		spaceRepo:                  spaceRepo,
		spaceMemberRepo:            spaceMemberRepo,
		oauthAuthorizationCodeRepo: oauthAuthorizationCodeRepo,
		createValidator:            createValidator,
	}
}

// CreateOAuthAuthorizationInputは同意の送信の入力パラメータ。Paramsは同意画面が引き継いだ
// 認可要求のパラメーターで、表示のときと同じ検証をやり直す。GrantedScopesは利用者が同意画面で
// 選んだ付与するスコープで、許可したときだけ検証する
type CreateOAuthAuthorizationInput struct {
	UserID        model.UserID
	Params        OAuthAuthorizationParams
	Approved      bool
	GrantedScopes []string
}

// CreateOAuthAuthorizationOutputは、認可コードを付けてクライアントへ戻すための値
type CreateOAuthAuthorizationOutput struct {
	RedirectURI string
	State       string

	// Codeは認可コードの値そのもの。データベースにはダイジェストしか残らない
	Code string
}

// Executeは、許可なら利用者が選んだスコープで許可を作成または置き換え、認可コードを発行して
// 返す。拒否した場合と、送信の時点でユーザーが連携先のスペースでアプリを許可できない場合は、
// access_deniedの*model.OAuthAuthorizationErrorを返す。選んだスコープが不正なら
// *model.ValidationErrorを返す。要求が不正な場合とフィーチャーフラグが無効な場合は
// GetOAuthAuthorizationNewUsecaseと同じ
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

	// 3. バリデーション
	scopes, err := uc.createValidator.Validate(ctx, validator.OAuthAuthorizationCreateValidatorInput{
		RequestedScopes: req.scopes,
		GrantedScopes:   input.GrantedScopes,
	})
	if err != nil {
		return nil, err
	}

	// 4. 永続化
	code, err := auth.GenerateOpaqueToken(auth.OAuthAuthorizationCodePrefix)
	if err != nil {
		return nil, fmt.Errorf("認可コードの生成に失敗: %w", err)
	}

	_, err = uc.createCodeWithGrant(ctx, repository.CreateOAuthAuthorizationCodeWithGrantInput{
		OAuthApplicationID: req.application.ID,
		SpaceID:            req.space.ID,
		SpaceMemberID:      decision.spaceMember.ID,
		CodeDigest:         auth.DigestOpaqueToken(code),
		Scopes:             scopes,
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

// createCodeWithGrantは、許可のロックを取得してから同意の範囲へ置き換える。
// ロックと失効を別の文にすることで、ロック待機中に発行されたトークンも失効の対象にする
func (uc *CreateOAuthAuthorizationUsecase) createCodeWithGrant(ctx context.Context, input repository.CreateOAuthAuthorizationCodeWithGrantInput) (*model.OAuthAuthorizationCode, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクション開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := uc.oauthGrantRepo.WithTx(tx).LockOrCreate(ctx, repository.CreateOAuthGrantInput{
		OAuthApplicationID: input.OAuthApplicationID, SpaceID: input.SpaceID, SpaceMemberID: input.SpaceMemberID, Scopes: input.Scopes,
	}); err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可のロックに失敗: %w", err)
	}
	code, err := uc.oauthAuthorizationCodeRepo.WithTx(tx).CreateWithGrant(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return code, nil
}
