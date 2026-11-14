package validator

import (
	"context"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// SettingsTwoFactorAuthCreateValidatorは、二要素認証を有効にするときの現在のパスワードと
// 認証アプリのコードを検証する
type SettingsTwoFactorAuthCreateValidator struct {
	userPasswordRepo *repository.UserPasswordRepository
}

// NewSettingsTwoFactorAuthCreateValidatorはSettingsTwoFactorAuthCreateValidatorを生成する
func NewSettingsTwoFactorAuthCreateValidator(userPasswordRepo *repository.UserPasswordRepository) *SettingsTwoFactorAuthCreateValidator {
	return &SettingsTwoFactorAuthCreateValidator{userPasswordRepo: userPasswordRepo}
}

// SettingsTwoFactorAuthCreateValidatorInputはバリデーションの入力パラメータ
type SettingsTwoFactorAuthCreateValidatorInput struct {
	UserID   model.UserID
	Password string
	TOTPCode string
	// Secretは、認証アプリへ登録してもらっている登録の途中の秘密鍵
	Secret string
}

// Validateは、現在のパスワードと、認証アプリのコードが登録の途中の秘密鍵に合っているかを確かめる。
// どちらも誤っていれば両方の項目に誤りを示す。
func (v *SettingsTwoFactorAuthCreateValidator) Validate(ctx context.Context, input SettingsTwoFactorAuthCreateValidatorInput) error {
	ve := model.NewValidationError()

	if input.Password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_required"))
	}
	if input.TOTPCode == "" {
		ve.AddField("totp_code", i18n.T(ctx, "validation_required"))
	} else if !totpCodeRegex.MatchString(input.TOTPCode) {
		ve.AddField("totp_code", i18n.T(ctx, "validation_totp_code_invalid"))
	}
	if ve.HasErrors() {
		return ve
	}

	userPassword, err := v.userPasswordRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		return err
	}
	if userPassword == nil || !auth.VerifyPassword(userPassword.PasswordDigest, input.Password) {
		ve.AddField("password", i18n.T(ctx, "validation_current_password_invalid"))
	}
	if !auth.ValidateTOTPCode(input.TOTPCode, input.Secret, time.Now()) {
		ve.AddField("totp_code", i18n.T(ctx, "validation_totp_code_incorrect"))
	}
	if ve.HasErrors() {
		return ve
	}

	return nil
}
