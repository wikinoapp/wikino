package sign_in_two_factor

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// TOTPコードは6桁の数字のみ
var totpCodeRegex = regexp.MustCompile(`^\d{6}$`)

// バリデーションのエラー定義
var (
	// ErrTwoFactorNotEnabled は2FAが有効でない場合のエラー
	ErrTwoFactorNotEnabled = errors.New("2FAが有効ではありません")
	// ErrInvalidTOTPCode はTOTPコードが無効な場合のエラー
	ErrInvalidTOTPCode = errors.New("TOTPコードが無効です")
)

// CreateValidator は2FAコード検証のバリデーションを行う
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
	UserID   model.UserID
	TOTPCode string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	FormErrors *session.FormErrors
	Err        error
}

// Validate はバリデーションを行う
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
	// 1. 形式バリデーション
	formErrors := session.NewFormErrors()

	if input.TOTPCode == "" {
		formErrors.AddField("totp_code", i18n.T(ctx, "validation_required"))
	} else if !totpCodeRegex.MatchString(input.TOTPCode) {
		formErrors.AddField("totp_code", i18n.T(ctx, "validation_totp_code_invalid"))
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

	// TOTPコードを検証
	// Rails版と同様に前後15秒のドリフトを許容
	valid := totp.Validate(input.TOTPCode, twoFactorAuth.Secret)
	if !valid {
		// ドリフトを考慮した検証
		valid = validateWithDrift(input.TOTPCode, twoFactorAuth.Secret, 15)
	}

	if !valid {
		formErrors.AddGlobal(i18n.T(ctx, "sign_in_two_factor_invalid_code"))
		return &CreateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrInvalidTOTPCode,
		}
	}

	// 検証成功
	return &CreateValidatorResult{}
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
