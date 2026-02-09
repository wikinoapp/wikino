package account

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
	// ErrEmailConfirmationNotFound はメール確認情報が見つからない場合のエラー
	ErrEmailConfirmationNotFound = errors.New("メール確認情報が見つかりません")
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

// CreateValidator はアカウント作成のバリデーションを行う
type CreateValidator struct {
	emailConfirmationRepo *repository.EmailConfirmationRepository
	userRepo              *repository.UserRepository
}

// NewCreateValidator は CreateValidator を生成する
func NewCreateValidator(
	emailConfirmationRepo *repository.EmailConfirmationRepository,
	userRepo *repository.UserRepository,
) *CreateValidator {
	return &CreateValidator{
		emailConfirmationRepo: emailConfirmationRepo,
		userRepo:              userRepo,
	}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
	EmailConfirmationID string
	Atname              string
	Password            string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
	EmailConfirmation *model.EmailConfirmation
	FormErrors        *session.FormErrors
	Err               error
}

// Validate はバリデーションを行う
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
	// 1. メール確認情報の取得と確認済みチェック（状態バリデーション）
	// これは形式バリデーションより先に行う（EmailConfirmationがないとフォームを表示できないため）
	emailConfirmation, err := v.emailConfirmationRepo.FindByID(ctx, input.EmailConfirmationID)
	if err != nil {
		return &CreateValidatorResult{Err: err}
	}
	if emailConfirmation == nil {
		return &CreateValidatorResult{Err: ErrEmailConfirmationNotFound}
	}

	// メール確認が完了しているかチェック
	if !emailConfirmation.IsSucceeded() {
		return &CreateValidatorResult{Err: ErrEmailNotConfirmed}
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
		return &CreateValidatorResult{
			EmailConfirmation: emailConfirmation,
			FormErrors:        formErrors,
		}
	}

	// 3. アットネームの重複チェック（状態バリデーション）
	existingUser, err := v.userRepo.FindByAtname(ctx, input.Atname)
	if err != nil {
		return &CreateValidatorResult{Err: err}
	}
	if existingUser != nil {
		formErrors.AddField("atname", i18n.T(ctx, "validation_atname_already_taken"))
		return &CreateValidatorResult{
			EmailConfirmation: emailConfirmation,
			FormErrors:        formErrors,
			Err:               ErrAtnameAlreadyTaken,
		}
	}

	// 検証成功
	return &CreateValidatorResult{EmailConfirmation: emailConfirmation}
}
