package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// UpdatePasswordResetUsecase はパスワードリセットによるパスワード更新ユースケース
type UpdatePasswordResetUsecase struct {
	db                     *sql.DB
	passwordResetTokenRepo *repository.PasswordResetTokenRepository
	userPasswordRepo       *repository.UserPasswordRepository
	updateValidator        *validator.PasswordUpdateValidator
}

// NewUpdatePasswordResetUsecase は UpdatePasswordResetUsecase を生成する
func NewUpdatePasswordResetUsecase(
	db *sql.DB,
	passwordResetTokenRepo *repository.PasswordResetTokenRepository,
	userPasswordRepo *repository.UserPasswordRepository,
	updateValidator *validator.PasswordUpdateValidator,
) *UpdatePasswordResetUsecase {
	return &UpdatePasswordResetUsecase{
		db:                     db,
		passwordResetTokenRepo: passwordResetTokenRepo,
		userPasswordRepo:       userPasswordRepo,
		updateValidator:        updateValidator,
	}
}

// UpdatePasswordResetInput はパスワード更新（リセット経由）の入力パラメータ
type UpdatePasswordResetInput struct {
	Token                string
	Password             string
	PasswordConfirmation string
}

// UpdatePasswordResetOutput はパスワード更新（リセット経由）の出力パラメータ
type UpdatePasswordResetOutput struct {
	UserID model.UserID
}

// Execute はバリデーション・パスワード更新・トークン使用済みマークを行う
func (uc *UpdatePasswordResetUsecase) Execute(ctx context.Context, input UpdatePasswordResetInput) (*UpdatePasswordResetOutput, error) {
	// 1. バリデーション
	validated, err := uc.updateValidator.Validate(ctx, validator.PasswordUpdateValidatorInput{
		Token:                input.Token,
		Password:             input.Password,
		PasswordConfirmation: input.PasswordConfirmation,
	})
	if err != nil {
		return nil, err
	}

	// 2. パスワードをハッシュ化（トランザクション前）
	passwordDigest, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("パスワードのハッシュ化に失敗しました: %w", err)
	}

	// 3. トランザクション（永続化のみ）
	return uc.updatePassword(ctx, validated, passwordDigest)
}

func (uc *UpdatePasswordResetUsecase) updatePassword(ctx context.Context, validated *validator.PasswordUpdateValidateOutput, passwordDigest string) (*UpdatePasswordResetOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	tokenRepo := uc.passwordResetTokenRepo.WithTx(tx)
	passwordRepo := uc.userPasswordRepo.WithTx(tx)

	if err := passwordRepo.UpdatePasswordDigest(ctx, validated.UserID, passwordDigest); err != nil {
		return nil, fmt.Errorf("パスワードの更新に失敗しました: %w", err)
	}

	if err := tokenRepo.MarkAsUsed(ctx, validated.TokenID); err != nil {
		return nil, fmt.Errorf("トークンの使用済みマークに失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &UpdatePasswordResetOutput{
		UserID: validated.UserID,
	}, nil
}
