package validator

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// personalAccessTokenNameMaxLengthはトークン名の最大文字数
const personalAccessTokenNameMaxLength = 50

// PersonalAccessTokenCreateValidatorは個人アクセストークンの発行のバリデーションを行う
type PersonalAccessTokenCreateValidator struct{}

// NewPersonalAccessTokenCreateValidatorはPersonalAccessTokenCreateValidatorを生成する
func NewPersonalAccessTokenCreateValidator() *PersonalAccessTokenCreateValidator {
	return &PersonalAccessTokenCreateValidator{}
}

// PersonalAccessTokenCreateValidatorInputはバリデーションの入力パラメータ。
// ScopesとExpirationDaysはフォームが送信した文字列で、変換はバリデーターが行う。
type PersonalAccessTokenCreateValidatorInput struct {
	Name           string
	Scopes         []string
	ExpirationDays string
}

// PersonalAccessTokenCreateValidateOutputはバリデーション成功時の出力
type PersonalAccessTokenCreateValidateOutput struct {
	// Scopesはトークンに付与するスコープ。重複を除き、model.APITokenScopesの順に並べる
	Scopes []model.Scope

	// ExpirationDaysは有効期限の日数。model.PersonalAccessTokenExpirationDaysのいずれかになる
	ExpirationDays int
}

// Validateはバリデーションを行い、フォームが送信したスコープと有効期限を変換して返す。
func (v *PersonalAccessTokenCreateValidator) Validate(ctx context.Context, input PersonalAccessTokenCreateValidatorInput) (*PersonalAccessTokenCreateValidateOutput, error) {
	ve := model.NewValidationError()

	validatePersonalAccessTokenName(ctx, ve, input.Name)
	scopes := validatePersonalAccessTokenScopes(ctx, ve, input.Scopes)
	expirationDays := validatePersonalAccessTokenExpirationDays(ctx, ve, input.ExpirationDays)

	if ve.HasErrors() {
		return nil, ve
	}

	return &PersonalAccessTokenCreateValidateOutput{
		Scopes:         scopes,
		ExpirationDays: expirationDays,
	}, nil
}

// validatePersonalAccessTokenNameはトークン名の形式を検証し、見つかったものをveに積む。
// 名前は一覧で持ち主がトークンを見分けるためのもので、一意である必要は無い。
func validatePersonalAccessTokenName(ctx context.Context, ve *model.ValidationError, name string) {
	if name == "" {
		ve.AddField("name", i18n.T(ctx, "validation_personal_access_token_name_required"))
		return
	}

	if utf8.RuneCountInString(name) > personalAccessTokenNameMaxLength {
		ve.AddField("name", i18n.T(ctx, "validation_personal_access_token_name_too_long"))
	}

	// 制御文字と先頭・末尾の空白を拒否する理由はトピック名と同じで、一覧に並んだときに
	// 読めない名前や、見分けの付かない名前にならないようにするためである。
	if strings.ContainsFunc(name, unicode.IsControl) {
		ve.AddField("name", i18n.T(ctx, "validation_personal_access_token_name_control_chars"))
	}
	if strings.HasPrefix(name, " ") || strings.HasSuffix(name, " ") {
		ve.AddField("name", i18n.T(ctx, "validation_personal_access_token_name_invalid_format"))
	}
}

// validatePersonalAccessTokenScopesは送信されたスコープを検証し、トークンに付与するスコープを
// 返す。トークンに付与できるスコープ (model.APITokenScopes) 以外が1つでも含まれていれば、
// 送信全体を拒否する。フォームには付与できるスコープしか出さないため、それ以外が届くのは
// 改ざんされた送信だけであり、黙って取り除くと利用者の意図と違うトークンを発行することになる。
func validatePersonalAccessTokenScopes(ctx context.Context, ve *model.ValidationError, values []string) []model.Scope {
	if len(values) == 0 {
		ve.AddField("scopes", i18n.T(ctx, "validation_personal_access_token_scopes_required"))
		return nil
	}

	for _, value := range values {
		if !model.HasScope(model.APITokenScopes, model.Scope(value)) {
			ve.AddField("scopes", i18n.T(ctx, "validation_personal_access_token_scopes_invalid"))
			return nil
		}
	}

	scopes := make([]model.Scope, 0, len(model.APITokenScopes))
	for _, scope := range model.APITokenScopes {
		if slices.Contains(values, string(scope)) {
			scopes = append(scopes, scope)
		}
	}
	return scopes
}

// validatePersonalAccessTokenExpirationDaysは送信された有効期限を検証し、日数を返す。
// 選択肢 (model.PersonalAccessTokenExpirationDays) 以外の値は拒否する。
func validatePersonalAccessTokenExpirationDays(ctx context.Context, ve *model.ValidationError, value string) int {
	days, err := strconv.Atoi(value)
	if err != nil || !slices.Contains(model.PersonalAccessTokenExpirationDays, days) {
		ve.AddField("expiration_days", i18n.T(ctx, "validation_personal_access_token_expiration_days_invalid"))
		return 0
	}
	return days
}
