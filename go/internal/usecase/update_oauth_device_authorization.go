package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// OAuthDeviceUserCodeFieldは、ユーザーコードを受け付けないエラーが付く項目名。HandlerはValidatorに
// 依存できないため、エラーの振り分けに使う定義をValidatorから引き継いで公開する
const OAuthDeviceUserCodeField = validator.OAuthDeviceUserCodeField

// UpdateOAuthDeviceAuthorizationUsecaseは、デバイス認可の同意画面での利用者の決定を受け、
// 許可待ちの要求を許可済みか拒否済みにする。
type UpdateOAuthDeviceAuthorizationUsecase struct {
	db                           *sql.DB
	featureFlagRepo              *repository.FeatureFlagRepository
	oauthDeviceAuthorizationRepo *repository.OAuthDeviceAuthorizationRepository
	oauthGrantRepo               *repository.OAuthGrantRepository
	spaceMemberRepo              *repository.SpaceMemberRepository
	verificationValidator        *validator.OAuthDeviceVerificationValidator
	createValidator              *validator.OAuthAuthorizationCreateValidator
}

// NewUpdateOAuthDeviceAuthorizationUsecaseはUpdateOAuthDeviceAuthorizationUsecaseを生成する
func NewUpdateOAuthDeviceAuthorizationUsecase(
	db *sql.DB,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthDeviceAuthorizationRepo *repository.OAuthDeviceAuthorizationRepository,
	oauthGrantRepo *repository.OAuthGrantRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	verificationValidator *validator.OAuthDeviceVerificationValidator,
	createValidator *validator.OAuthAuthorizationCreateValidator,
) *UpdateOAuthDeviceAuthorizationUsecase {
	return &UpdateOAuthDeviceAuthorizationUsecase{
		db:                           db,
		featureFlagRepo:              featureFlagRepo,
		oauthDeviceAuthorizationRepo: oauthDeviceAuthorizationRepo,
		oauthGrantRepo:               oauthGrantRepo,
		spaceMemberRepo:              spaceMemberRepo,
		verificationValidator:        verificationValidator,
		createValidator:              createValidator,
	}
}

// UpdateOAuthDeviceAuthorizationInputは同意の送信の入力パラメータ。UserCodeは同意画面が引き継いだ
// ユーザーコードで、表示のときと同じ照合をやり直す。GrantedScopesは利用者が選んだ付与するスコープで、
// 許可したときだけ検証する
type UpdateOAuthDeviceAuthorizationInput struct {
	UserID        model.UserID
	UserCode      string
	Approved      bool
	GrantedScopes []string
}

// UpdateOAuthDeviceAuthorizationOutputは利用者の決定の結果
type UpdateOAuthDeviceAuthorizationOutput struct {
	Application *model.OAuthApplication

	// Approvedは要求を許可したか。偽なら拒否した
	Approved bool

	// DeniedReasonは、許可を選んだが送信の時点でユーザーが連携先のスペースでアプリを許可できない
	// 理由。空でなければ要求を変えておらず、呼び出し側は理由を示して選び直させる
	DeniedReason OAuthAuthorizationDeniedReason
}

// Executeは、許可なら利用者が選んだスコープで許可を新しく作り、要求を許可済みにする。
// 拒否なら要求を拒否済みにする。ユーザーコードを受け付けない場合と、選んだスコープが不正な場合は
// *model.ValidationErrorを、フィーチャーフラグが無効なら*model.AppErrorを返す
func (uc *UpdateOAuthDeviceAuthorizationUsecase) Execute(ctx context.Context, input UpdateOAuthDeviceAuthorizationInput) (*UpdateOAuthDeviceAuthorizationOutput, error) {
	// 1. データ取得と検証
	if err := checkPublicAPIEnabled(ctx, uc.featureFlagRepo, input.UserID); err != nil {
		return nil, err
	}

	v, err := uc.verificationValidator.Validate(ctx, validator.OAuthDeviceVerificationValidatorInput{UserCode: input.UserCode})
	if err != nil {
		return nil, err
	}

	// 拒否はアプリに権限を与えないため、許可できない利用者にも認める (同意画面の代わりに理由を
	// 示す画面からも拒否を送れる)
	if !input.Approved {
		denied, err := uc.oauthDeviceAuthorizationRepo.Deny(ctx, v.DeviceAuthorization.ID, v.Space.ID)
		if err != nil {
			return nil, fmt.Errorf("デバイス認可の要求の拒否に失敗: %w", err)
		}
		if denied == nil {
			return nil, validator.NewInvalidOAuthDeviceUserCodeError(ctx)
		}
		return &UpdateOAuthDeviceAuthorizationOutput{Application: v.Application}, nil
	}

	// 2. 認可チェック。同意画面を出した後にメンバーから外されたり権限を失ったりしていないかを、
	// 送信の時点で確かめ直す
	decision, err := decideOAuthAuthorization(ctx, uc.spaceMemberRepo, v.Space, input.UserID)
	if err != nil {
		return nil, err
	}
	if decision.denied != "" {
		return &UpdateOAuthDeviceAuthorizationOutput{Application: v.Application, DeniedReason: decision.denied}, nil
	}

	// 3. バリデーション
	scopes, err := uc.createValidator.Validate(ctx, validator.OAuthAuthorizationCreateValidatorInput{
		RequestedScopes: v.DeviceAuthorization.Scopes,
		GrantedScopes:   input.GrantedScopes,
	})
	if err != nil {
		return nil, err
	}

	// 4. 永続化
	approved, err := uc.approveWithGrant(ctx, repository.ApproveOAuthDeviceAuthorizationWithGrantInput{
		ID:                 v.DeviceAuthorization.ID,
		OAuthApplicationID: v.Application.ID,
		SpaceID:            v.Space.ID,
		SpaceMemberID:      decision.spaceMember.ID,
		Scopes:             scopes,
	})
	if err != nil {
		return nil, err
	}
	if approved == nil {
		return nil, validator.NewInvalidOAuthDeviceUserCodeError(ctx)
	}
	return &UpdateOAuthDeviceAuthorizationOutput{Application: v.Application, Approved: true}, nil
}

// approveWithGrantは、要求のロックを取得し、同じアプリ・メンバーの許可を上限に収まるよう押し出してから、
// 選んだ範囲で許可を作って要求を許可済みにする。
// 要求のロックは、同じ要求への同時の許可・拒否とデバイスコードの交換を直列化する。ロック取得後に
// 許可待ちでなければ、または要求のロック・許可の作成の待機中に期限が切れたら、
// 変更をロールバックしてnilを返す
func (uc *UpdateOAuthDeviceAuthorizationUsecase) approveWithGrant(ctx context.Context, input repository.ApproveOAuthDeviceAuthorizationWithGrantInput) (*model.OAuthDeviceAuthorization, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクション開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	deviceAuthorizationRepo := uc.oauthDeviceAuthorizationRepo.WithTx(tx)
	pending, err := deviceAuthorizationRepo.LockPending(ctx, input.ID, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("デバイス認可の要求のロックに失敗: %w", err)
	}
	now := time.Now()
	if pending == nil || pending.IsExpired(now) {
		return nil, nil
	}
	if err := uc.oauthGrantRepo.WithTx(tx).Evict(ctx, repository.EvictInput{
		OAuthApplicationID: input.OAuthApplicationID, SpaceID: input.SpaceID, SpaceMemberID: input.SpaceMemberID, Now: now,
	}); err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可の押し出しに失敗: %w", err)
	}
	approved, err := deviceAuthorizationRepo.ApproveWithGrant(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("デバイス認可の要求の許可に失敗: %w", err)
	}
	// 許可の作成も外部キーの検査で待機しうるため、コミット前に期限を確かめ直す。
	// 期限切れなら要求の更新と許可の作成をどちらもロールバックする。
	if approved == nil || approved.IsExpired(time.Now()) {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return approved, nil
}
