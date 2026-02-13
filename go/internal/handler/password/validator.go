package password

import (
	"context"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// バリデーションのエラー定義
var (
	ErrTokenNotFound = errors.New("トークンが見つかりません")
	ErrTokenUsed     = errors.New("トークンは既に使用されています")
	ErrTokenExpired  = errors.New("トークンの有効期限が切れています")
)

// UpdateValidator はパスワード更新のバリデーションを行う
type UpdateValidator struct {
	passwordResetTokenRepo *repository.PasswordResetTokenRepository
}

// NewUpdateValidator は UpdateValidator を生成する
func NewUpdateValidator(passwordResetTokenRepo *repository.PasswordResetTokenRepository) *UpdateValidator {
	return &UpdateValidator{
		passwordResetTokenRepo: passwordResetTokenRepo,
	}
}

// UpdateValidatorInput はバリデーションの入力パラメータ
type UpdateValidatorInput struct {
	Token                string
	Password             string
	PasswordConfirmation string
}

// UpdateValidatorResult はバリデーションの結果
type UpdateValidatorResult struct {
	// TokenID は検証成功時のトークンID（UseCaseに渡す）
	TokenID string
	// UserID は検証成功時のユーザーID（UseCaseに渡す）
	UserID string
	// FormErrors はフォームエラー
	FormErrors *session.FormErrors
	// Err はシステムエラー
	Err error
}

const minPasswordLength = 8

// Validate はバリデーションを行う
func (v *UpdateValidator) Validate(ctx context.Context, input UpdateValidatorInput) *UpdateValidatorResult {
	formErrors := session.NewFormErrors()

	// 形式バリデーション
	// トークン必須チェック
	if input.Token == "" {
		formErrors.AddGlobal(i18n.T(ctx, "validation_token_required"))
		return &UpdateValidatorResult{FormErrors: formErrors}
	}

	// パスワード必須チェック
	if input.Password == "" {
		formErrors.AddField("password", i18n.T(ctx, "validation_password_required"))
	}

	// パスワード確認必須チェック
	if input.PasswordConfirmation == "" {
		formErrors.AddField("password_confirmation", i18n.T(ctx, "validation_password_confirmation_required"))
	}

	// パスワード文字数チェック
	if len(input.Password) > 0 && len(input.Password) < minPasswordLength {
		formErrors.AddField("password", i18n.T(ctx, "validation_password_too_short"))
	}

	// パスワード確認一致チェック
	if input.Password != "" && input.PasswordConfirmation != "" && input.Password != input.PasswordConfirmation {
		formErrors.AddField("password_confirmation", i18n.T(ctx, "validation_password_confirmation_mismatch"))
	}

	if formErrors.HasErrors() {
		return &UpdateValidatorResult{FormErrors: formErrors}
	}

	// トークン検証（状態バリデーション）
	token, err := v.validateToken(ctx, input.Token)
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenNotFound):
			formErrors.AddGlobal(i18n.T(ctx, "validation_token_invalid"))
			return &UpdateValidatorResult{FormErrors: formErrors, Err: err}
		case errors.Is(err, ErrTokenUsed):
			formErrors.AddGlobal(i18n.T(ctx, "validation_token_used"))
			return &UpdateValidatorResult{FormErrors: formErrors, Err: err}
		case errors.Is(err, ErrTokenExpired):
			formErrors.AddGlobal(i18n.T(ctx, "validation_token_expired"))
			return &UpdateValidatorResult{FormErrors: formErrors, Err: err}
		default:
			// DBエラーはログに記録し、ユーザーにはシステムエラーを表示
			formErrors.AddGlobal(i18n.T(ctx, "validation_system_error"))
			return &UpdateValidatorResult{FormErrors: formErrors, Err: err}
		}
	}

	return &UpdateValidatorResult{
		TokenID: token.ID,
		UserID:  token.UserID,
	}
}

// validateToken はトークンの検証を行う
func (v *UpdateValidator) validateToken(ctx context.Context, token string) (*model.PasswordResetToken, error) {
	tokenDigest := password_reset.HashToken(token)
	tokenModel, err := v.passwordResetTokenRepo.FindByTokenDigest(ctx, tokenDigest)
	if err != nil {
		return nil, err
	}

	if tokenModel == nil {
		return nil, ErrTokenNotFound
	}

	if tokenModel.IsUsed() {
		return nil, ErrTokenUsed
	}

	if tokenModel.IsExpired() {
		return nil, ErrTokenExpired
	}

	return tokenModel, nil
}
