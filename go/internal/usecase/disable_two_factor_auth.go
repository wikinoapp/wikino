package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// DisableTwoFactorAuthUsecaseは、現在のパスワードを照合して二要素認証を無効にする
type DisableTwoFactorAuthUsecase struct {
	deleteValidator       *validator.SettingsTwoFactorAuthDeleteValidator
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewDisableTwoFactorAuthUsecaseはDisableTwoFactorAuthUsecaseを生成する
func NewDisableTwoFactorAuthUsecase(
	deleteValidator *validator.SettingsTwoFactorAuthDeleteValidator,
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *DisableTwoFactorAuthUsecase {
	return &DisableTwoFactorAuthUsecase{
		deleteValidator:       deleteValidator,
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// DisableTwoFactorAuthInputはDisableTwoFactorAuthUsecase.Executeの入力
type DisableTwoFactorAuthInput struct {
	UserID   model.UserID
	Password string
}

// Executeは現在のパスワードを照合し、二要素認証を無効にしてリカバリーコードを空にする。
//
// パスワードの誤りは*model.ValidationErrorで返す。
// 二要素認証が有効でないとき (二重送信で既に無効にした・別の画面で無効にした) は、
// AppErrCodeConflictの*model.AppErrorを返す。
// 無効にするのは1つの更新のため、トランザクションを開かない。
func (uc *DisableTwoFactorAuthUsecase) Execute(ctx context.Context, input DisableTwoFactorAuthInput) error {
	enabled, err := uc.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("二要素認証の設定の取得に失敗: %w", err)
	}
	if enabled == nil {
		return notEnabledTwoFactorAuthError(ctx, input.UserID)
	}

	if err := uc.deleteValidator.Validate(ctx, validator.SettingsTwoFactorAuthDeleteValidatorInput{
		UserID:   input.UserID,
		Password: input.Password,
	}); err != nil {
		return err
	}

	disabled, err := uc.userTwoFactorAuthRepo.Disable(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("二要素認証の無効化に失敗: %w", err)
	}
	if !disabled {
		return notEnabledTwoFactorAuthError(ctx, input.UserID)
	}

	return nil
}
