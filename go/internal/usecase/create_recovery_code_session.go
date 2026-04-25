package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateRecoveryCodeSessionUsecase はリカバリーコード認証によるセッション作成ユースケース
type CreateRecoveryCodeSessionUsecase struct {
	db                    *sql.DB
	recoveryValidator     *validator.SignInTwoFactorRecoveryCreateValidator
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
	userSessionRepo       *repository.UserSessionRepository
}

// NewCreateRecoveryCodeSessionUsecase は CreateRecoveryCodeSessionUsecase を生成する
func NewCreateRecoveryCodeSessionUsecase(
	db *sql.DB,
	recoveryValidator *validator.SignInTwoFactorRecoveryCreateValidator,
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
	userSessionRepo *repository.UserSessionRepository,
) *CreateRecoveryCodeSessionUsecase {
	return &CreateRecoveryCodeSessionUsecase{
		db:                    db,
		recoveryValidator:     recoveryValidator,
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
		userSessionRepo:       userSessionRepo,
	}
}

// CreateRecoveryCodeSessionInput はリカバリーコードセッション作成の入力パラメータ
type CreateRecoveryCodeSessionInput struct {
	UserID       model.UserID
	RecoveryCode string
	IPAddress    string
	UserAgent    string
}

// CreateRecoveryCodeSessionOutput はリカバリーコードセッション作成の出力パラメータ
type CreateRecoveryCodeSessionOutput struct {
	Token string
}

// Execute はリカバリーコードを検証・消費してセッションを作成する
func (uc *CreateRecoveryCodeSessionUsecase) Execute(ctx context.Context, input CreateRecoveryCodeSessionInput) (*CreateRecoveryCodeSessionOutput, error) {
	// 1. バリデーション（形式チェック + リカバリーコード検証）
	twoFactorAuth, err := uc.validate(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. トランザクション（リカバリーコード消費 + セッション作成）
	return uc.persist(ctx, input, twoFactorAuth.RecoveryCodes)
}

func (uc *CreateRecoveryCodeSessionUsecase) validate(ctx context.Context, input CreateRecoveryCodeSessionInput) (*model.UserTwoFactorAuth, error) {
	return uc.recoveryValidator.Validate(ctx, validator.SignInTwoFactorRecoveryCreateValidatorInput{
		UserID:       input.UserID,
		RecoveryCode: input.RecoveryCode,
	})
}

func (uc *CreateRecoveryCodeSessionUsecase) persist(ctx context.Context, input CreateRecoveryCodeSessionInput, currentCodes []string) (*CreateRecoveryCodeSessionOutput, error) {
	token, err := auth.GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("セッショントークンの生成に失敗しました: %w", err)
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	userTwoFactorAuthRepo := uc.userTwoFactorAuthRepo.WithTx(tx)
	userSessionRepo := uc.userSessionRepo.WithTx(tx)

	// リカバリーコードを消費
	newCodes := removeRecoveryCode(currentCodes, input.RecoveryCode)
	if err := userTwoFactorAuthRepo.UpdateRecoveryCodes(ctx, input.UserID, newCodes); err != nil {
		return nil, fmt.Errorf("リカバリーコードの消費に失敗しました: %w", err)
	}

	// セッションを作成
	now := time.Now()
	_, err = userSessionRepo.Create(ctx, repository.CreateInput{
		UserID:     input.UserID,
		Token:      token,
		IPAddress:  input.IPAddress,
		UserAgent:  input.UserAgent,
		SignedInAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("セッションの作成に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &CreateRecoveryCodeSessionOutput{
		Token: token,
	}, nil
}
