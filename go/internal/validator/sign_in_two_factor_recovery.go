package validator

import (
	"context"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// リカバリーコードは8文字の英小文字と数字のみ
var recoveryCodeRegex = regexp.MustCompile(`^[a-z0-9]{8}$`)

// SignInTwoFactorRecoveryCreateValidator はリカバリーコード認証のバリデーションを行う
type SignInTwoFactorRecoveryCreateValidator struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewSignInTwoFactorRecoveryCreateValidator は SignInTwoFactorRecoveryCreateValidator を生成する
func NewSignInTwoFactorRecoveryCreateValidator(
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *SignInTwoFactorRecoveryCreateValidator {
	return &SignInTwoFactorRecoveryCreateValidator{
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// SignInTwoFactorRecoveryCreateValidatorInput はバリデーションの入力パラメータ
type SignInTwoFactorRecoveryCreateValidatorInput struct {
	UserID       model.UserID
	RecoveryCode string
}

// Validate はバリデーションを行う
func (v *SignInTwoFactorRecoveryCreateValidator) Validate(ctx context.Context, input SignInTwoFactorRecoveryCreateValidatorInput) (*model.UserTwoFactorAuth, error) {
	// 1. 形式バリデーション
	ve := model.NewValidationError()

	if input.RecoveryCode == "" {
		ve.AddField("recovery_code", i18n.T(ctx, "validation_required"))
	} else if !recoveryCodeRegex.MatchString(input.RecoveryCode) {
		ve.AddField("recovery_code", i18n.T(ctx, "validation_recovery_code_invalid"))
	}

	if ve.HasErrors() {
		return nil, ve
	}

	// 2. 状態バリデーション（DB検証）
	twoFactorAuth, err := v.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if twoFactorAuth == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeTwoFactorNotEnabled,
			UserMsg: i18n.T(ctx, "error_two_factor_not_enabled"),
		}
	}

	// リカバリーコードを検証
	if !isValidRecoveryCode(twoFactorAuth, input.RecoveryCode) {
		ve.AddGlobal(i18n.T(ctx, "sign_in_two_factor_recovery_invalid_code"))
		return nil, ve
	}

	// 検証成功
	return twoFactorAuth, nil
}

// isValidRecoveryCode はリカバリーコードが有効かどうかを確認する
func isValidRecoveryCode(twoFactorAuth *model.UserTwoFactorAuth, code string) bool {
	for _, recoveryCode := range twoFactorAuth.RecoveryCodes {
		if recoveryCode == code {
			return true
		}
	}
	return false
}
