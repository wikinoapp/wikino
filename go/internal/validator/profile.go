package validator

import (
	"context"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// プロフィールのバリデーション定数
const (
	UserNameMaxLength        = 30
	UserDescriptionMaxLength = 150
)

// ProfileUpdateValidatorはプロフィールの編集のバリデーションを行う
type ProfileUpdateValidator struct {
	userRepo *repository.UserRepository
}

// NewProfileUpdateValidatorはProfileUpdateValidatorを生成する
func NewProfileUpdateValidator(userRepo *repository.UserRepository) *ProfileUpdateValidator {
	return &ProfileUpdateValidator{userRepo: userRepo}
}

// ProfileUpdateValidatorInputはバリデーションの入力パラメータ
type ProfileUpdateValidatorInput struct {
	UserID      model.UserID
	Atname      string
	Name        string
	Description string
}

// Validateはバリデーションを行う
func (v *ProfileUpdateValidator) Validate(ctx context.Context, input ProfileUpdateValidatorInput) error {
	ve := model.NewValidationError()

	addAtnameFormatErrors(ctx, ve, input.Atname)

	if utf8.RuneCountInString(input.Name) > UserNameMaxLength {
		ve.AddField("name", i18n.T(ctx, "validation_user_name_too_long"))
	}
	if utf8.RuneCountInString(input.Description) > UserDescriptionMaxLength {
		ve.AddField("description", i18n.T(ctx, "validation_user_description_too_long"))
	}

	if ve.HasErrors() {
		return ve
	}

	return validateAtnameAvailability(ctx, v.userRepo, input.Atname, &input.UserID)
}
