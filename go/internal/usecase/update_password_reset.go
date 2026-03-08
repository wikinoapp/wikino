package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// UpdatePasswordResetUsecase はパスワードリセットによるパスワード更新ユースケース
type UpdatePasswordResetUsecase struct {
	db                     *sql.DB
	passwordResetTokenRepo *repository.PasswordResetTokenRepository
	userPasswordRepo       *repository.UserPasswordRepository
}

// NewUpdatePasswordResetUsecase は UpdatePasswordResetUsecase を生成する
func NewUpdatePasswordResetUsecase(
	db *sql.DB,
	passwordResetTokenRepo *repository.PasswordResetTokenRepository,
	userPasswordRepo *repository.UserPasswordRepository,
) *UpdatePasswordResetUsecase {
	return &UpdatePasswordResetUsecase{
		db:                     db,
		passwordResetTokenRepo: passwordResetTokenRepo,
		userPasswordRepo:       userPasswordRepo,
	}
}

// UpdatePasswordResetInput はパスワード更新（リセット経由）の入力パラメータ
// トークンの検証（存在、有効期限、使用済み）はハンドラーの validator.go で行う
type UpdatePasswordResetInput struct {
	TokenID     string
	UserID      model.UserID
	NewPassword string
}

// UpdatePasswordResetOutput はパスワード更新（リセット経由）の出力パラメータ
type UpdatePasswordResetOutput struct {
	UserID model.UserID
}

// Execute はパスワードを更新し、トークンを使用済みにマークする
func (uc *UpdatePasswordResetUsecase) Execute(ctx context.Context, input UpdatePasswordResetInput) (*UpdatePasswordResetOutput, error) {
	// パスワードをハッシュ化
	passwordDigest, err := auth.HashPassword(input.NewPassword)
	if err != nil {
		return nil, fmt.Errorf("パスワードのハッシュ化に失敗しました: %w", err)
	}

	// トランザクションを開始
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// トランザクション内で操作するためのリポジトリを取得
	tokenRepo := uc.passwordResetTokenRepo.WithTx(tx)
	passwordRepo := uc.userPasswordRepo.WithTx(tx)

	// パスワードを更新
	if err := passwordRepo.UpdatePasswordDigest(ctx, input.UserID, passwordDigest); err != nil {
		return nil, fmt.Errorf("パスワードの更新に失敗しました: %w", err)
	}

	// トークンを使用済みにマーク
	if err := tokenRepo.MarkAsUsed(ctx, input.TokenID); err != nil {
		return nil, fmt.Errorf("トークンの使用済みマークに失敗しました: %w", err)
	}

	// トランザクションをコミット
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &UpdatePasswordResetOutput{
		UserID: input.UserID,
	}, nil
}
