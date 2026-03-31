// Package validator はバリデーションを提供します
package validator

import (
	"context"
	"net/mail"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// SignInCreateValidator はサインインのバリデーションを行う
type SignInCreateValidator struct {
	userRepo              *repository.UserRepository
	userPasswordRepo      *repository.UserPasswordRepository
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
}

// NewSignInCreateValidator は SignInCreateValidator を生成する
func NewSignInCreateValidator(
	userRepo *repository.UserRepository,
	userPasswordRepo *repository.UserPasswordRepository,
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
) *SignInCreateValidator {
	return &SignInCreateValidator{
		userRepo:              userRepo,
		userPasswordRepo:      userPasswordRepo,
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
	}
}

// SignInCreateValidatorInput はバリデーションの入力パラメータ
type SignInCreateValidatorInput struct {
	Email    string
	Password string
}

// SignInCreateValidateOutput はバリデーション成功時の出力
type SignInCreateValidateOutput struct {
	User              *model.User
	UserTwoFactorAuth *model.UserTwoFactorAuth
}

// Validate はバリデーションを行う
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) (*SignInCreateValidateOutput, error) {
	// 1. 形式バリデーション
	ve := model.NewValidationError()

	if input.Email == "" {
		ve.AddField("email", i18n.T(ctx, "validation_required"))
	} else if !isValidEmail(input.Email) {
		ve.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	if input.Password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_required"))
	}

	if ve.HasErrors() {
		return nil, ve
	}

	// 2. 状態バリデーション（DB検証）
	user, err := v.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		// セキュリティ対策: 存在しないメールアドレスでも同じエラーメッセージを表示
		ve.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		return nil, ve
	}

	// パスワードを取得
	userPassword, err := v.userPasswordRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if userPassword == nil {
		// パスワードが設定されていない場合も同じエラーメッセージを表示
		ve.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		return nil, ve
	}

	// パスワードを検証
	if !auth.VerifyPassword(userPassword.PasswordDigest, input.Password) {
		ve.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		return nil, ve
	}

	// 二要素認証の有効化状態を取得
	twoFactorAuth, err := v.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &SignInCreateValidateOutput{User: user, UserTwoFactorAuth: twoFactorAuth}, nil
}

// isValidEmail はメールアドレスの形式をチェックします
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
