package validator

import (
	"context"
	"regexp"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// TOTPコードは6桁の数字のみ
var totpCodeRegex = regexp.MustCompile(`^\d{6}$`)

// SignInTwoFactorCreateValidatorは2FAコード検証のバリデーションを行う
type SignInTwoFactorCreateValidator struct {
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewSignInTwoFactorCreateValidatorはSignInTwoFactorCreateValidatorを生成する
func NewSignInTwoFactorCreateValidator(
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *SignInTwoFactorCreateValidator {
	return &SignInTwoFactorCreateValidator{
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// SignInTwoFactorCreateValidatorInputはバリデーションの入力パラメータ
type SignInTwoFactorCreateValidatorInput struct {
	UserID   model.UserID
	TOTPCode string
}

// Validateはバリデーションを行う
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

	// 2. 状態バリデーション (DB検証)
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

	// 二要素認証の有効化と同じ許容幅で照合する
	valid := auth.ValidateTOTPCode(input.TOTPCode, twoFactorAuth.Secret, time.Now())

	if !valid {
		ve.AddGlobal(i18n.T(ctx, "sign_in_two_factor_invalid_code"))
		return ve
	}

	// 検証成功
	return nil
}
