package validator

import (
	"context"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// アットネームのバリデーション定数
const (
	AtnameMaxLength = 20
)

// アットネームの形式（英数字とアンダースコアのみ）
var atnameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// パスワードのバリデーション定数
const (
	PasswordMinLength = 8
)

// AccountCreateValidator はアカウント作成のバリデーションを行う
type AccountCreateValidator struct {
	userRepo *repository.UserRepository
}

// NewAccountCreateValidator は AccountCreateValidator を生成する
func NewAccountCreateValidator(
	userRepo *repository.UserRepository,
) *AccountCreateValidator {
	return &AccountCreateValidator{
		userRepo: userRepo,
	}
}

// AccountCreateValidatorInput はバリデーションの入力パラメータ
type AccountCreateValidatorInput struct {
	Atname   string
	Password string
}

// Validate はバリデーションを行う
func (v *AccountCreateValidator) Validate(ctx context.Context, input AccountCreateValidatorInput) error {
	ve := model.NewValidationError()

	// アットネームのバリデーション
	if input.Atname == "" {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_required"))
	} else {
		if len(input.Atname) > AtnameMaxLength {
			ve.AddField("atname", i18n.T(ctx, "validation_atname_too_long"))
		}
		if !atnameRegex.MatchString(input.Atname) {
			ve.AddField("atname", i18n.T(ctx, "validation_atname_invalid_format"))
		}
	}

	// パスワードのバリデーション
	if input.Password == "" {
		ve.AddField("password", i18n.T(ctx, "validation_password_required"))
	} else if len(input.Password) < PasswordMinLength {
		ve.AddField("password", i18n.T(ctx, "validation_password_too_short"))
	}

	if ve.HasErrors() {
		return ve
	}

	// アットネームの重複チェック（状態バリデーション）
	existingUser, err := v.userRepo.FindByAtname(ctx, input.Atname)
	if err != nil {
		return err
	}
	if existingUser != nil {
		ve.AddField("atname", i18n.T(ctx, "validation_atname_already_taken"))
		return ve
	}

	return nil
}
