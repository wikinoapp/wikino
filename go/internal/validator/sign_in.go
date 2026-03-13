// Package validator はバリデーションを提供します
package validator

import (
	"context"
	"errors"
	"net/mail"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// バリデーションのエラー定義
var (
	// ErrUserNotFound はユーザーが見つからない場合のエラー
	ErrUserNotFound = errors.New("ユーザーが見つかりません")
	// ErrPasswordNotSet はパスワードが設定されていない場合のエラー
	ErrPasswordNotSet = errors.New("パスワードが設定されていません")
	// ErrInvalidPassword はパスワードが一致しない場合のエラー
	ErrInvalidPassword = errors.New("パスワードが正しくありません")
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

// SignInCreateValidatorResult はバリデーションの結果
type SignInCreateValidatorResult struct {
	User              *model.User
	UserTwoFactorAuth *model.UserTwoFactorAuth
	FormErrors        *session.FormErrors
	Err               error
}

// Validate はバリデーションを行う
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) *SignInCreateValidatorResult {
	// 1. 形式バリデーション
	formErrors := session.NewFormErrors()

	if input.Email == "" {
		formErrors.AddField("email", i18n.T(ctx, "validation_required"))
	} else if !isValidEmail(input.Email) {
		formErrors.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	if input.Password == "" {
		formErrors.AddField("password", i18n.T(ctx, "validation_required"))
	}

	if formErrors.HasErrors() {
		return &SignInCreateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	user, err := v.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return &SignInCreateValidatorResult{Err: err}
	}
	if user == nil {
		// セキュリティ対策: 存在しないメールアドレスでも同じエラーメッセージを表示
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		return &SignInCreateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrUserNotFound,
		}
	}

	// パスワードを取得
	userPassword, err := v.userPasswordRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return &SignInCreateValidatorResult{Err: err}
	}
	if userPassword == nil {
		// パスワードが設定されていない場合も同じエラーメッセージを表示
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		return &SignInCreateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrPasswordNotSet,
		}
	}

	// パスワードを検証
	if !auth.VerifyPassword(userPassword.PasswordDigest, input.Password) {
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		return &SignInCreateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrInvalidPassword,
		}
	}

	// 二要素認証の有効化状態を取得
	twoFactorAuth, err := v.userTwoFactorAuthRepo.FindEnabledByUserID(ctx, user.ID)
	if err != nil {
		return &SignInCreateValidatorResult{Err: err}
	}

	// 検証成功
	return &SignInCreateValidatorResult{User: user, UserTwoFactorAuth: twoFactorAuth}
}

// isValidEmail はメールアドレスの形式をチェックします
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
