package validator

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// PasswordUpdateValidator はパスワード更新のバリデーションを行う
type PasswordUpdateValidator struct {
	passwordResetTokenRepo *repository.PasswordResetTokenRepository
}

// NewPasswordUpdateValidator は PasswordUpdateValidator を生成する
func NewPasswordUpdateValidator(passwordResetTokenRepo *repository.PasswordResetTokenRepository) *PasswordUpdateValidator {
	return &PasswordUpdateValidator{
		passwordResetTokenRepo: passwordResetTokenRepo,
	}
}

// PasswordUpdateValidatorInput はバリデーションの入力パラメータ
type PasswordUpdateValidatorInput struct {
	Token                string
	Password             string
	PasswordConfirmation string
}

// PasswordUpdateValidateOutput はバリデーション成功時の出力
type PasswordUpdateValidateOutput struct {
	TokenID string
	UserID  model.UserID
}

// Validate はバリデーションを行う
func (v *PasswordUpdateValidator) Validate(ctx context.Context, input PasswordUpdateValidatorInput) (*PasswordUpdateValidateOutput, error) {
	ve := model.NewValidationError()

	// 形式バリデーション
	// トークン必須チェック
	if input.Token == "" {
		ve.AddGlobal(i18n.T(ctx, "validation_token_required"))
		return nil, ve
	}

	// パスワード必須チェック・強度チェック
	if input.Password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_password_required"))
	} else {
		addPasswordStrengthError(ctx, ve, input.Password)
	}

	// パスワード確認必須チェック
	if input.PasswordConfirmation == "" {
		ve.AddField("password_confirmation", i18n.T(ctx, "validation_password_confirmation_required"))
	}

	// パスワード確認一致チェック
	if input.Password != "" && input.PasswordConfirmation != "" && input.Password != input.PasswordConfirmation {
		ve.AddField("password_confirmation", i18n.T(ctx, "validation_password_confirmation_mismatch"))
	}

	if ve.HasErrors() {
		return nil, ve
	}

	// トークン検証（状態バリデーション）
	token, err := v.validateToken(ctx, input.Token)
	if err != nil {
		return nil, err
	}

	return &PasswordUpdateValidateOutput{
		TokenID: token.ID,
		UserID:  token.UserID,
	}, nil
}

// validateToken はトークンの検証を行う
func (v *PasswordUpdateValidator) validateToken(ctx context.Context, token string) (*model.PasswordResetToken, error) {
	tokenDigest := password_reset.HashToken(token)
	tokenModel, err := v.passwordResetTokenRepo.FindByTokenDigest(ctx, tokenDigest)
	if err != nil {
		return nil, err
	}

	ve := model.NewValidationError()

	if tokenModel == nil {
		ve.AddGlobal(i18n.T(ctx, "validation_token_invalid"))
		return nil, ve
	}

	if tokenModel.IsUsed() {
		ve.AddGlobal(i18n.T(ctx, "validation_token_used"))
		return nil, ve
	}

	if tokenModel.IsExpired() {
		ve.AddGlobal(i18n.T(ctx, "validation_token_expired"))
		return nil, ve
	}

	return tokenModel, nil
}
