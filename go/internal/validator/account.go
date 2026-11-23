package validator

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// AccountCreateValidatorはアカウント作成のバリデーションを行う
type AccountCreateValidator struct {
	userRepo *repository.UserRepository
}

// NewAccountCreateValidatorはAccountCreateValidatorを生成する
func NewAccountCreateValidator(
	userRepo *repository.UserRepository,
) *AccountCreateValidator {
	return &AccountCreateValidator{
		userRepo: userRepo,
	}
}

// AccountCreateValidatorInputはバリデーションの入力パラメータ
type AccountCreateValidatorInput struct {
	Atname   string
	Password string
}

// Validateはバリデーションを行う
func (v *AccountCreateValidator) Validate(ctx context.Context, input AccountCreateValidatorInput) error {
	ve := model.NewValidationError()

	addAtnameFormatErrors(ctx, ve, input.Atname)

	// パスワードのバリデーション
	if input.Password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_password_required"))
	} else {
		addPasswordStrengthError(ctx, ve, input.Password)
	}

	if ve.HasErrors() {
		return ve
	}

	return validateAtnameAvailability(ctx, v.userRepo, input.Atname, nil)
}
