package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/repository"
)

// VerifyEmailConfirmationUsecase はメール確認コード検証ユースケース
type VerifyEmailConfirmationUsecase struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
}

// NewVerifyEmailConfirmationUsecase は VerifyEmailConfirmationUsecase を生成する
func NewVerifyEmailConfirmationUsecase(
	emailConfirmationRepo *repository.EmailConfirmationRepository,
) *VerifyEmailConfirmationUsecase {
	return &VerifyEmailConfirmationUsecase{
		emailConfirmationRepo: emailConfirmationRepo,
	}
}

// VerifyEmailConfirmationInput はメール確認コード検証の入力パラメータ
type VerifyEmailConfirmationInput struct {
	EmailConfirmationID string
	Code                string
}

// エラー定義
var (
	// ErrEmailConfirmationNotFound は確認情報が見つからない場合のエラー
	ErrEmailConfirmationNotFound = errors.New("メール確認情報が見つかりません")
	// ErrEmailConfirmationAlreadySucceeded は既に確認済みの場合のエラー
	ErrEmailConfirmationAlreadySucceeded = errors.New("このメール確認は既に完了しています")
	// ErrEmailConfirmationExpired は有効期限切れの場合のエラー
	ErrEmailConfirmationExpired = errors.New("確認コードの有効期限が切れています")
	// ErrEmailConfirmationCodeMismatch はコードが一致しない場合のエラー
	ErrEmailConfirmationCodeMismatch = errors.New("確認コードが正しくありません")
)

// Execute はメール確認コードを検証し、確認を完了状態に更新する
func (uc *VerifyEmailConfirmationUsecase) Execute(ctx context.Context, input VerifyEmailConfirmationInput) error {
	// ID でメール確認情報を取得
	confirmation, err := uc.emailConfirmationRepo.FindByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return err
	}
	if confirmation == nil {
		return ErrEmailConfirmationNotFound
	}

	// 既に確認済みの場合はエラー
	if confirmation.IsSucceeded() {
		return ErrEmailConfirmationAlreadySucceeded
	}

	// 有効期限チェック（15分）
	if confirmation.IsExpired() {
		return ErrEmailConfirmationExpired
	}

	// コードの一致チェック（大文字小文字を区別しない）
	if !strings.EqualFold(confirmation.Code, input.Code) {
		return ErrEmailConfirmationCodeMismatch
	}

	// 確認完了状態に更新
	return uc.emailConfirmationRepo.Succeed(ctx, input.EmailConfirmationID)
}
