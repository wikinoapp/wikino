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
			GrantedScopes:   []string{"page:read", "topic:read", "page:read"},
		})
		if err != nil {
			t.Fatalf("Validate()のエラー = %v", err)
		}
		if want := []model.Scope{model.ScopeTopicRead, model.ScopePageRead}; !slices.Equal(got, want) {
			t.Errorf("Validate() = %v、期待値 = %v", got, want)
		}
	})

	t.Run("選んだスコープが含むスコープのうち、要求にあるものを補う", func(t *testing.T) {
		t.Parallel()

		// 同意画面は含まれるスコープを無効にして送信しないため、送られなくても補う
		got, err := v.Validate(ctx, validator.OAuthAuthorizationCreateValidatorInput{
			RequestedScopes: []model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageWrite},
			GrantedScopes:   []string{"draft_page:write"},
		})
		if err != nil {
			t.Fatalf("Validate()のエラー = %v", err)
		}
		if want := []model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageWrite}; !slices.Equal(got, want) {
			t.Errorf("Validate() = %v、期待値 = %v", got, want)
		}
	})

	t.Run("アプリが要求していないスコープは補わない", func(t *testing.T) {
		t.Parallel()

		got, err := v.Validate(ctx, validator.OAuthAuthorizationCreateValidatorInput{
			RequestedScopes: []model.Scope{model.ScopeTopicRead, model.ScopeDraftPageWrite},
			GrantedScopes:   []string{"draft_page:write"},
		})
		if err != nil {
			t.Fatalf("Validate()のエラー = %v", err)
		}
		if want := []model.Scope{model.ScopeDraftPageWrite}; !slices.Equal(got, want) {
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
