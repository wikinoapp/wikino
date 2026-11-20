package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTwoFactorAuthRecoveryCodesUsecaseは、二要素認証が有効な間だけ、リカバリーコードを取得する
type GetTwoFactorAuthRecoveryCodesUsecase struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewGetTwoFactorAuthRecoveryCodesUsecaseはGetTwoFactorAuthRecoveryCodesUsecaseを生成する
func NewGetTwoFactorAuthRecoveryCodesUsecase(userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository) *GetTwoFactorAuthRecoveryCodesUsecase {
	return &GetTwoFactorAuthRecoveryCodesUsecase{userTwoFactorAuthRepo: userTwoFactorAuthRepo}
}

// GetTwoFactorAuthRecoveryCodesInputはGetTwoFactorAuthRecoveryCodesUsecase.Executeの入力
type GetTwoFactorAuthRecoveryCodesInput struct {
	UserID model.UserID
}

// GetTwoFactorAuthRecoveryCodesOutputはGetTwoFactorAuthRecoveryCodesUsecase.Executeの結果
type GetTwoFactorAuthRecoveryCodesOutput struct {
	// Enabledは二要素認証が有効か。有効でないときRecoveryCodesは空
	Enabled bool
	// RecoveryCodesは残っている (まだ使っていない) リカバリーコード
	RecoveryCodes []string
}

// Executeは二要素認証が有効なら、残っているリカバリーコードを返す
func (uc *GetTwoFactorAuthRecoveryCodesUsecase) Execute(ctx context.Context, input GetTwoFactorAuthRecoveryCodesInput) (*GetTwoFactorAuthRecoveryCodesOutput, error) {
	twoFactorAuth, err := uc.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("二要素認証の設定の取得に失敗: %w", err)
	}
	if twoFactorAuth == nil {
		return &GetTwoFactorAuthRecoveryCodesOutput{}, nil
	}

	return &GetTwoFactorAuthRecoveryCodesOutput{Enabled: true, RecoveryCodes: twoFactorAuth.RecoveryCodes}, nil
}
