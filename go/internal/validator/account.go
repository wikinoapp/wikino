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

// IsValidAtname reports whether the atname satisfies the format and the length
// every account's atname has to have. It is exported so that a caller creating
// an account outside the sign-up form checks the same rule instead of a copy
// of it.
//
// AccountCreateValidator applies the same two constraints separately, because
// the form tells the two apart in what it says. A rule added here has to be
// added there as well, or the form would accept an atname the roster refuses.
//
// [Ja] IsValidAtname は、アットネームがすべてのアカウントに課される形式と長さを
// 満たすかを返す。サインアップフォームの外でアカウントを作る呼び出し元が、制約の
// 写しではなく同じ規則を参照できるよう公開している。
//
// AccountCreateValidator は同じ 2 つの制約を個別に適用する。フォームは 2 つを
// 別のメッセージで伝え分けるため。ここへ規則を足したときは、あちらへも足す必要が
// ある。そうしないと、フォームが受理する atname を名簿が拒否することになる。
func IsValidAtname(atname string) bool {
	return len(atname) <= AtnameMaxLength && atnameRegex.MatchString(atname)
}

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
	} else {
		addPasswordStrengthError(ctx, ve, input.Password)
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
