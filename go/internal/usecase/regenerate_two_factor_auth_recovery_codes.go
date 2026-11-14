package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// RegenerateTwoFactorAuthRecoveryCodesUsecaseは、現在のパスワードを照合して二要素認証のリカバリーコードを作り直す
type RegenerateTwoFactorAuthRecoveryCodesUsecase struct {
	createValidator       *validator.SettingsTwoFactorAuthRecoveryCodeCreateValidator
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewRegenerateTwoFactorAuthRecoveryCodesUsecaseはRegenerateTwoFactorAuthRecoveryCodesUsecaseを生成する
func NewRegenerateTwoFactorAuthRecoveryCodesUsecase(
	createValidator *validator.SettingsTwoFactorAuthRecoveryCodeCreateValidator,
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *RegenerateTwoFactorAuthRecoveryCodesUsecase {
	return &RegenerateTwoFactorAuthRecoveryCodesUsecase{
		createValidator:       createValidator,
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// RegenerateTwoFactorAuthRecoveryCodesInputはRegenerateTwoFactorAuthRecoveryCodesUsecase.Executeの入力
type RegenerateTwoFactorAuthRecoveryCodesInput struct {
	UserID   model.UserID
	Password string
}

// Executeは現在のパスワードを照合し、リカバリーコードを新しい10個に置き換える。前のコードは使えなくなる。
//
// パスワードの誤りは*model.ValidationErrorで返す。
// 二要素認証が有効でないときは、AppErrCodeConflictの*model.AppErrorを返す。
// 置き換えるのは1つの更新のため、トランザクションを開かない。
func (uc *RegenerateTwoFactorAuthRecoveryCodesUsecase) Execute(ctx context.Context, input RegenerateTwoFactorAuthRecoveryCodesInput) error {
	enabled, err := uc.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("二要素認証の設定の取得に失敗: %w", err)
	}
	if enabled == nil {
		return notEnabledTwoFactorAuthError(ctx, input.UserID)
	}

	if err := uc.createValidator.Validate(ctx, validator.SettingsTwoFactorAuthRecoveryCodeCreateValidatorInput{
		UserID:   input.UserID,
		Password: input.Password,
	}); err != nil {
		return err
	}

	recoveryCodes, err := auth.GenerateRecoveryCodes()
	if err != nil {
		return fmt.Errorf("リカバリーコードの生成に失敗: %w", err)
	}

	regenerated, err := uc.userTwoFactorAuthRepo.RegenerateRecoveryCodes(ctx, input.UserID, recoveryCodes)
	if err != nil {
		return fmt.Errorf("リカバリーコードの再生成に失敗: %w", err)
	}
	if !regenerated {
		return notEnabledTwoFactorAuthError(ctx, input.UserID)
	}

	return nil
}
