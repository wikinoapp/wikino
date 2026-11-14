package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// errTwoFactorAuthNotEnabledは、操作しようとした時点で二要素認証が有効でないことを表す内部エラー
var errTwoFactorAuthNotEnabled = errors.New("二要素認証が有効ではありません")

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

// notEnabledTwoFactorAuthErrorは、二要素認証が有効でないことを表すAppErrCodeConflictの*model.AppErrorを返す。
// ハンドラーはこのエラーで二要素認証の設定の画面へ送り、今の状態を見せる。
func notEnabledTwoFactorAuthError(ctx context.Context, userID model.UserID) *model.AppError {
	return &model.AppError{
		Code:     model.AppErrCodeConflict,
		UserMsg:  i18n.T(ctx, "flash_two_factor_auth_not_enabled"),
		Internal: errTwoFactorAuthNotEnabled,
		Metadata: map[string]string{"user_id": userID.String()},
	}
}
