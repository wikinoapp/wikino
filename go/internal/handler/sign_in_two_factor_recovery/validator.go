package sign_in_two_factor_recovery

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
	// ErrTwoFactorNotEnabled は2FAが有効でない場合のエラー
	ErrTwoFactorNotEnabled = errors.New("2FAが有効ではありません")
	// ErrInvalidRecoveryCode はリカバリーコードが無効な場合のエラー
	ErrInvalidRecoveryCode = errors.New("リカバリーコードが無効です")
)

// CreateValidator はリカバリーコード認証のバリデーションを行う
type CreateValidator struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewCreateValidator は CreateValidator を生成する
func NewCreateValidator(
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *CreateValidator {
	return &CreateValidator{
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	UserID       string
	RecoveryCode string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	TwoFactorAuth *model.UserTwoFactorAuth
	FormErrors    *session.FormErrors
	Err           error
}

// Validate はバリデーションを行う
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
	// 1. 形式バリデーション
	formErrors := session.NewFormErrors()

	if input.RecoveryCode == "" {
		formErrors.AddField("recovery_code", i18n.T(ctx, "validation_required"))
	} else if !recoveryCodeRegex.MatchString(input.RecoveryCode) {
		formErrors.AddField("recovery_code", i18n.T(ctx, "validation_recovery_code_invalid"))
	}

	if formErrors.HasErrors() {
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	twoFactorAuth, err := v.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, input.UserID)
	if err != nil {
		return &CreateValidatorResult{Err: err}
	}
	if twoFactorAuth == nil {
		return &CreateValidatorResult{Err: ErrTwoFactorNotEnabled}
	}

	// リカバリーコードを検証
	if !isValidRecoveryCode(twoFactorAuth, input.RecoveryCode) {
		formErrors.AddGlobal(i18n.T(ctx, "sign_in_two_factor_recovery_invalid_code"))
		return &CreateValidatorResult{
			TwoFactorAuth: twoFactorAuth,
			FormErrors:    formErrors,
			Err:           ErrInvalidRecoveryCode,
		}
	}

	// 検証成功
	return &CreateValidatorResult{
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
