package validator

import (
	"context"
	"errors"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// リカバリーコードは8文字の英小文字と数字のみ
var recoveryCodeRegex = regexp.MustCompile(`^[a-z0-9]{8}$`)

// バリデーションのエラー定義
var (
	// ErrInvalidRecoveryCode はリカバリーコードが無効な場合のエラー
	ErrInvalidRecoveryCode = errors.New("リカバリーコードが無効です")
)

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

// SignInTwoFactorRecoveryCreateValidatorResult はバリデーションの結果
type SignInTwoFactorRecoveryCreateValidatorResult struct {
	TwoFactorAuth *model.UserTwoFactorAuth
	FormErrors    *session.FormErrors
	Err           error
}

// Validate はバリデーションを行う
func (v *SignInTwoFactorRecoveryCreateValidator) Validate(ctx context.Context, input SignInTwoFactorRecoveryCreateValidatorInput) *SignInTwoFactorRecoveryCreateValidatorResult {
	// 1. 形式バリデーション
	formErrors := session.NewFormErrors()

	if input.RecoveryCode == "" {
		formErrors.AddField("recovery_code", i18n.T(ctx, "validation_required"))
	} else if !recoveryCodeRegex.MatchString(input.RecoveryCode) {
		formErrors.AddField("recovery_code", i18n.T(ctx, "validation_recovery_code_invalid"))
	}

	if formErrors.HasErrors() {
		return &SignInTwoFactorRecoveryCreateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	twoFactorAuth, err := v.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, input.UserID)
	if err != nil {
		return &SignInTwoFactorRecoveryCreateValidatorResult{Err: err}
	}
	if twoFactorAuth == nil {
		return &SignInTwoFactorRecoveryCreateValidatorResult{Err: ErrTwoFactorNotEnabled}
	}

	// リカバリーコードを検証
	if !isValidRecoveryCode(twoFactorAuth, input.RecoveryCode) {
		formErrors.AddGlobal(i18n.T(ctx, "sign_in_two_factor_recovery_invalid_code"))
		return &SignInTwoFactorRecoveryCreateValidatorResult{
			TwoFactorAuth: twoFactorAuth,
			FormErrors:    formErrors,
			Err:           ErrInvalidRecoveryCode,
		}
	}

	// 検証成功
	return &SignInTwoFactorRecoveryCreateValidatorResult{
		TwoFactorAuth: twoFactorAuth,
	}
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
