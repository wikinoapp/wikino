package validator

import (
	"context"
	"regexp"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// TOTPコードは6桁の数字のみ
var totpCodeRegex = regexp.MustCompile(`^\d{6}$`)

// SignInTwoFactorCreateValidator は2FAコード検証のバリデーションを行う
type SignInTwoFactorCreateValidator struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewSignInTwoFactorCreateValidator は SignInTwoFactorCreateValidator を生成する
func NewSignInTwoFactorCreateValidator(
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *SignInTwoFactorCreateValidator {
	return &SignInTwoFactorCreateValidator{
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// SignInTwoFactorCreateValidatorInput はバリデーションの入力パラメータ
type SignInTwoFactorCreateValidatorInput struct {
	UserID   model.UserID
	TOTPCode string
}

// Validate はバリデーションを行う
func (v *SignInTwoFactorCreateValidator) Validate(ctx context.Context, input SignInTwoFactorCreateValidatorInput) error {
	// 1. 形式バリデーション
	ve := model.NewValidationError()

	if input.TOTPCode == "" {
		ve.AddField("totp_code", i18n.T(ctx, "validation_required"))
	} else if !totpCodeRegex.MatchString(input.TOTPCode) {
		ve.AddField("totp_code", i18n.T(ctx, "validation_totp_code_invalid"))
	}

	if ve.HasErrors() {
		return ve
	}

	// 2. 状態バリデーション（DB検証）
	twoFactorAuth, err := v.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, input.UserID)
	if err != nil {
		return err
	}
	if twoFactorAuth == nil {
		return &model.AppError{
			Code:    model.AppErrCodeTwoFactorNotEnabled,
			UserMsg: i18n.T(ctx, "error_two_factor_not_enabled"),
		}
	}

	// TOTPコードを検証
	// Rails版と同様に前後15秒のドリフトを許容
	valid := totp.Validate(input.TOTPCode, twoFactorAuth.Secret)
	if !valid {
		// ドリフトを考慮した検証
		valid = validateWithDrift(input.TOTPCode, twoFactorAuth.Secret, 15)
	}

	if !valid {
		ve.AddGlobal(i18n.T(ctx, "sign_in_two_factor_invalid_code"))
		return ve
	}

	// 検証成功
	return nil
}

// validateWithDrift は前後のタイムステップも考慮してTOTPコードを検証する
func validateWithDrift(code, secret string, driftSeconds int) bool {
	now := time.Now()

	// 現在時刻で検証
	if totp.Validate(code, secret) {
		return true
	}

	// 前のタイムステップで検証
	opts := totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    6,
		Algorithm: 0, // SHA1 (default)
	}

	// 前後のタイムステップで検証
	pastTime := now.Add(-time.Duration(driftSeconds) * time.Second)
	if valid, _ := totp.ValidateCustom(code, secret, pastTime, opts); valid {
		return true
	}

	futureTime := now.Add(time.Duration(driftSeconds) * time.Second)
	if valid, _ := totp.ValidateCustom(code, secret, futureTime, opts); valid {
		return true
	}

	return false
}
