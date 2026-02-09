package sign_in

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

// CreateValidator はサインインのバリデーションを行う
type CreateValidator struct {
	userRepo         *repository.UserRepository
	userPasswordRepo *repository.UserPasswordRepository
}

// NewCreateValidator は CreateValidator を生成する
func NewCreateValidator(
	userRepo *repository.UserRepository,
	userPasswordRepo *repository.UserPasswordRepository,
) *CreateValidator {
	return &CreateValidator{
		userRepo:         userRepo,
		userPasswordRepo: userPasswordRepo,
	}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	Email    string
	Password string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	User       *model.User
	FormErrors *session.FormErrors
	Err        error
}

// Validate はバリデーションを行う
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
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
		return &CreateValidatorResult{FormErrors: formErrors}
	}

	// 2. 状態バリデーション（DB検証）
	user, err := v.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return &CreateValidatorResult{Err: err}
	}
	if user == nil {
		// セキュリティ対策: 存在しないメールアドレスでも同じエラーメッセージを表示
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		return &CreateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrUserNotFound,
		}
	}

	// パスワードを取得
	userPassword, err := v.userPasswordRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return &CreateValidatorResult{Err: err}
	}
	if userPassword == nil {
		// パスワードが設定されていない場合も同じエラーメッセージを表示
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		return &CreateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrPasswordNotSet,
		}
	}

	// パスワードを検証
	if !auth.VerifyPassword(userPassword.PasswordDigest, input.Password) {
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		return &CreateValidatorResult{
			FormErrors: formErrors,
			Err:        ErrInvalidPassword,
		}
	}

	// 検証成功
	return &CreateValidatorResult{User: user}
}

// isValidEmail はメールアドレスの形式をチェックします
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
