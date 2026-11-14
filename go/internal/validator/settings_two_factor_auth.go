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

// SettingsTwoFactorAuthDeleteValidatorは、二要素認証を無効にするときの現在のパスワードを検証する
type SettingsTwoFactorAuthDeleteValidator struct {
	userPasswordRepo *repository.UserPasswordRepository
}

// NewSettingsTwoFactorAuthDeleteValidatorはSettingsTwoFactorAuthDeleteValidatorを生成する
func NewSettingsTwoFactorAuthDeleteValidator(userPasswordRepo *repository.UserPasswordRepository) *SettingsTwoFactorAuthDeleteValidator {
	return &SettingsTwoFactorAuthDeleteValidator{userPasswordRepo: userPasswordRepo}
}

// SettingsTwoFactorAuthDeleteValidatorInputはバリデーションの入力パラメータ
type SettingsTwoFactorAuthDeleteValidatorInput struct {
	UserID   model.UserID
	Password string
}

// Validateは現在のパスワードを確かめる
func (v *SettingsTwoFactorAuthDeleteValidator) Validate(ctx context.Context, input SettingsTwoFactorAuthDeleteValidatorInput) error {
	return validateCurrentPassword(ctx, v.userPasswordRepo, input.UserID, input.Password)
}

// SettingsTwoFactorAuthRecoveryCodeCreateValidatorは、リカバリーコードを作り直すときの現在のパスワードを検証する
type SettingsTwoFactorAuthRecoveryCodeCreateValidator struct {
	userPasswordRepo *repository.UserPasswordRepository
}

// NewSettingsTwoFactorAuthRecoveryCodeCreateValidatorはSettingsTwoFactorAuthRecoveryCodeCreateValidatorを生成する
func NewSettingsTwoFactorAuthRecoveryCodeCreateValidator(userPasswordRepo *repository.UserPasswordRepository) *SettingsTwoFactorAuthRecoveryCodeCreateValidator {
	return &SettingsTwoFactorAuthRecoveryCodeCreateValidator{userPasswordRepo: userPasswordRepo}
}

// SettingsTwoFactorAuthRecoveryCodeCreateValidatorInputはバリデーションの入力パラメータ
type SettingsTwoFactorAuthRecoveryCodeCreateValidatorInput struct {
	UserID   model.UserID
	Password string
}

// Validateは現在のパスワードを確かめる
func (v *SettingsTwoFactorAuthRecoveryCodeCreateValidator) Validate(ctx context.Context, input SettingsTwoFactorAuthRecoveryCodeCreateValidatorInput) error {
	return validateCurrentPassword(ctx, v.userPasswordRepo, input.UserID, input.Password)
}

// validateCurrentPasswordは、passwordがユーザーの現在のパスワードかを確かめ、
// 空か誤りならpasswordの項目のValidationErrorを返す
func validateCurrentPassword(ctx context.Context, userPasswordRepo *repository.UserPasswordRepository, userID model.UserID, password string) error {
	ve := model.NewValidationError()

	if password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_required"))
		return ve
	}

	userPassword, err := userPasswordRepo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if userPassword == nil || !auth.VerifyPassword(userPassword.PasswordDigest, password) {
		ve.AddField("password", i18n.T(ctx, "validation_current_password_invalid"))
		return ve
	}

	return nil
}
