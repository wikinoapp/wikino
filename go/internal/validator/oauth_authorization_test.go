package validator_test

import (
	"context"
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestOAuthAuthorizationCreateValidator(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	v := validator.NewOAuthAuthorizationCreateValidator()
	requested := []model.Scope{model.ScopeTopicRead, model.ScopePageRead, model.ScopePageWrite}

	t.Run("要求の一部を選ぶと、選んだスコープを要求の順に重複を除いて返す", func(t *testing.T) {
		t.Parallel()

		got, err := v.Validate(ctx, validator.OAuthAuthorizationCreateValidatorInput{
			RequestedScopes: requested,
			GrantedScopes:   []string{"page:write", "topic:read", "page:write"},
		})
		if err != nil {
			t.Fatalf("Validate()のエラー = %v", err)
		}
		if want := []model.Scope{model.ScopeTopicRead, model.ScopePageWrite}; !slices.Equal(got, want) {
			t.Errorf("Validate() = %v、期待値 = %v", got, want)
		}
	})

	tests := []struct {
		name    string
		granted []string
		wantMsg string
	}{
		{name: "1つも選ばなければ拒否する", granted: nil, wantMsg: "許可する操作を1つ以上選んでください"},
		{name: "要求に無いスコープを含めば拒否する", granted: []string{"page:read", "draft_page:read"}, wantMsg: "要求されていない操作が含まれています"},
		{name: "存在しないスコープを含めば拒否する", granted: []string{"space:admin"}, wantMsg: "要求されていない操作が含まれています"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := v.Validate(ctx, validator.OAuthAuthorizationCreateValidatorInput{
				RequestedScopes: requested,
				GrantedScopes:   tt.granted,
			})
			if got != nil {
				t.Errorf("Validate() = %v、期待値 = nil", got)
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、ValidationErrorを期待", err)
			}
			if errs := ve.GetFieldErrors("granted_scopes"); !slices.Equal(errs, []string{tt.wantMsg}) {
				t.Errorf("granted_scopesのエラー = %v、期待値 = %v", errs, []string{tt.wantMsg})
			}
		})
	}
}
