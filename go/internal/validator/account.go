package validator

import (
	"context"
	"errors"
	"regexp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// バリデーションのエラー定義
var (
	// ErrEmailNotConfirmed はメール確認が完了していない場合のエラー
	ErrEmailNotConfirmed = errors.New("メール確認が完了していません")
	// ErrAtnameAlreadyTaken はアットネームが既に使用されている場合のエラー
	ErrAtnameAlreadyTaken = errors.New("このアットネームは既に使用されています")
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
	emailConfirmationRepo *repository.EmailConfirmationRepository
	userRepo              *repository.UserRepository
}

// NewAccountCreateValidator は AccountCreateValidator を生成する
func NewAccountCreateValidator(
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	userRepo *repository.UserRepository,
) *AccountCreateValidator {
	return &AccountCreateValidator{
		emailConfirmationRepo: emailConfirmationRepo,
		userRepo:              userRepo,
	}
}

// AccountCreateValidatorInput はバリデーションの入力パラメータ
type AccountCreateValidatorInput struct {
	EmailConfirmationID string
	Atname              string
	Password            string
}

// AccountCreateValidatorResult はバリデーションの結果
type AccountCreateValidatorResult struct {
	EmailConfirmation *model.EmailConfirmation
	FormErrors        *session.FormErrors
	Err               error
}

// Validate はバリデーションを行う
func (v *AccountCreateValidator) Validate(ctx context.Context, input AccountCreateValidatorInput) *AccountCreateValidatorResult {
	// 1. メール確認情報の取得と確認済みチェック（状態バリデーション）
	// これは形式バリデーションより先に行う（EmailConfirmationがないとフォームを表示できないため）
	emailConfirmation, err := v.emailConfirmationRepo.FindByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return &AccountCreateValidatorResult{Err: err}
	}
	if emailConfirmation == nil {
		return &AccountCreateValidatorResult{Err: ErrEmailConfirmationNotFound}
	}

	// メール確認が完了しているかチェック
	if !emailConfirmation.IsSucceeded() {
		return &AccountCreateValidatorResult{Err: ErrEmailNotConfirmed}
	}

	// 2. 形式バリデーション
	formErrors := session.NewFormErrors()

	// アットネームのバリデーション
	if input.Atname == "" {
		formErrors.AddField("atname", i18n.T(ctx, "validation_atname_required"))
	} else {
		if len(input.Atname) > AtnameMaxLength {
			formErrors.AddField("atname", i18n.T(ctx, "validation_atname_too_long"))
		}
		if !atnameRegex.MatchString(input.Atname) {
			formErrors.AddField("atname", i18n.T(ctx, "validation_atname_invalid_format"))
		}
	}

	// パスワードのバリデーション
	if input.Password == "" {
		formErrors.AddField("password", i18n.T(ctx, "validation_password_required"))
	} else if len(input.Password) < PasswordMinLength {
		formErrors.AddField("password", i18n.T(ctx, "validation_password_too_short"))
	}

	if formErrors.HasErrors() {
		return &AccountCreateValidatorResult{
			EmailConfirmation: emailConfirmation,
			FormErrors:        formErrors,
		}
	}

	// 3. アットネームの重複チェック（状態バリデーション）
	existingUser, err := v.userRepo.FindByAtname(ctx, input.Atname)
	if err != nil {
		return &AccountCreateValidatorResult{Err: err}
	}
	if existingUser != nil {
		formErrors.AddField("atname", i18n.T(ctx, "validation_atname_already_taken"))
		return &AccountCreateValidatorResult{
			EmailConfirmation: emailConfirmation,
			FormErrors:        formErrors,
			Err:               ErrAtnameAlreadyTaken,
		}
	}

	// 検証成功
	return &AccountCreateValidatorResult{EmailConfirmation: emailConfirmation}
}
