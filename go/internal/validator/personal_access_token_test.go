package validator_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestPersonalAccessTokenCreateValidator_Valid(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	v := validator.NewPersonalAccessTokenCreateValidator()

	tests := []struct {
		name           string
		tokenName      string
		scopes         []string
		expirationDays string
		wantScopes     []model.Scope
		wantDays       int
	}{
		{
			name:           "付与できるスコープを1つ選んだ",
			tokenName:      "CLI",
			scopes:         []string{"page:read"},
			expirationDays: "30",
			wantScopes:     []model.Scope{model.ScopePageRead},
			wantDays:       30,
		},
		{
			// フォームの並びや重複に左右されず、同じ集合なら同じ並びで保存する。
			name:           "スコープの重複を除きAPITokenScopesの順に並べる",
			tokenName:      strings.Repeat("あ", 50),
			scopes:         []string{"page:write", "topic:read", "page:write", "page:read"},
			expirationDays: "365",
			wantScopes:     []model.Scope{model.ScopeTopicRead, model.ScopePageRead, model.ScopePageWrite},
			wantDays:       365,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := v.Validate(ctx, validator.PersonalAccessTokenCreateValidatorInput{
				Name:           tt.tokenName,
				Scopes:         tt.scopes,
				ExpirationDays: tt.expirationDays,
			})
			if err != nil {
				t.Fatalf("Validate()のエラー = %v", err)
			}
			if !slices.Equal(output.Scopes, tt.wantScopes) {
				t.Errorf("Scopes = %v、期待値 = %v", output.Scopes, tt.wantScopes)
			}
			if output.ExpirationDays != tt.wantDays {
				t.Errorf("ExpirationDays = %d、期待値 = %d", output.ExpirationDays, tt.wantDays)
			}
		})
	}
}

func TestPersonalAccessTokenCreateValidator_Invalid(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	v := validator.NewPersonalAccessTokenCreateValidator()

	valid := validator.PersonalAccessTokenCreateValidatorInput{
		Name:           "CLI",
		Scopes:         []string{"page:read"},
		ExpirationDays: "30",
	}

	tests := []struct {
		name   string
		modify func(*validator.PersonalAccessTokenCreateValidatorInput)
		field  string
	}{
		{name: "名前が空", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.Name = "" }, field: "name"},
		{name: "名前が50文字を超える", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.Name = strings.Repeat("あ", 51) }, field: "name"},
		{name: "名前に改行を含む", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.Name = "foo\nbar" }, field: "name"},
		{name: "名前の先頭が空白", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.Name = " CLI" }, field: "name"},
		{name: "名前の末尾が空白", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.Name = "CLI " }, field: "name"},
		{name: "スコープを選んでいない", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.Scopes = nil }, field: "scopes"},
		{name: "管理系のスコープを含む", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) {
			i.Scopes = []string{"page:read", "space:admin"}
		}, field: "scopes"},
		{name: "トークン管理のスコープを含む", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) {
			i.Scopes = []string{"personal_access_token:write"}
		}, field: "scopes"},
		{name: "未知のスコープを含む", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.Scopes = []string{"page:admin"} }, field: "scopes"},
		{name: "有効期限が空", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.ExpirationDays = "" }, field: "expiration_days"},
		{name: "有効期限が選択肢に無い日数", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.ExpirationDays = "0" }, field: "expiration_days"},
		{name: "有効期限が数値でない", modify: func(i *validator.PersonalAccessTokenCreateValidatorInput) { i.ExpirationDays = "never" }, field: "expiration_days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := valid
			tt.modify(&input)

			output, err := v.Validate(ctx, input)
			if output != nil {
				t.Errorf("Validate()の出力 = %v、期待値 = nil", output)
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("Validate()のエラー = %v、期待値 = ValidationError", err)
			}
			if !ve.HasFieldError(tt.field) {
				t.Errorf("%sのフィールドエラーが無い", tt.field)
			}
		})
	}
}
