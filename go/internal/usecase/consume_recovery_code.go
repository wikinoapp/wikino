package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ConsumeRecoveryCodeUsecase はリカバリーコード消費ユースケース
// リカバリーコードをリストから削除する（DB更新を伴うため）
type ConsumeRecoveryCodeUsecase struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewConsumeRecoveryCodeUsecase は ConsumeRecoveryCodeUsecase を生成する
func NewConsumeRecoveryCodeUsecase(
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *ConsumeRecoveryCodeUsecase {
	return &ConsumeRecoveryCodeUsecase{
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// ConsumeRecoveryCodeInput はリカバリーコード消費の入力パラメータ
type ConsumeRecoveryCodeInput struct {
	UserID       string
	RecoveryCode string
	CurrentCodes []string
}

// Execute はリカバリーコードを消費する（リストから削除）
func (uc *ConsumeRecoveryCodeUsecase) Execute(ctx context.Context, input ConsumeRecoveryCodeInput) error {
	// リカバリーコードを削除
	newRecoveryCodes := removeRecoveryCode(input.CurrentCodes, input.RecoveryCode)
	return uc.userTwoFactorAuthRepo.UpdateRecoveryCodes(ctx, input.UserID, newRecoveryCodes)
}

// removeRecoveryCode はリカバリーコードをリストから削除する
func removeRecoveryCode(codes []string, codeToRemove string) []string {
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		if code != codeToRemove {
			result = append(result, code)
		}
	}
	return result
}
