package usecase

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestGetAPICurrentTokenUsecase_Execute(t *testing.T) {
	t.Parallel()

	expiresAt := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	principal := &model.APIPrincipal{
		User:        &model.User{ID: "user-1"},
		Space:       &model.Space{ID: "space-1", Identifier: "bound-space"},
		SpaceMember: &model.SpaceMember{ID: "space-member-1", SpaceID: "space-1", UserID: "user-1"},
		TokenKind:   model.APITokenKindOAuthAccessToken,
		// 認証で含意を展開した順のまま渡す
		TokenExpiresAt: expiresAt,
		Scopes:         []model.Scope{model.ScopeDraftPageWrite, model.ScopePageRead, model.ScopeDraftPageRead},
	}

	t.Run("トークンの種類・スコープ・有効期限を返す", func(t *testing.T) {
		t.Parallel()

		output, err := NewGetAPICurrentTokenUsecase().Execute(context.Background(), GetAPICurrentTokenInput{
			Principal:       principal,
			SpaceIdentifier: "bound-space",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Kind != model.APITokenKindOAuthAccessToken {
			t.Errorf("Kind = %s、期待値 = %s", output.Kind, model.APITokenKindOAuthAccessToken)
		}
		// スコープはAPITokenScopesの順に並べ直す
		wantScopes := []model.Scope{model.ScopePageRead, model.ScopeDraftPageRead, model.ScopeDraftPageWrite}
		if !slices.Equal(output.Scopes, wantScopes) {
			t.Errorf("Scopes = %v、期待値 = %v", output.Scopes, wantScopes)
		}
		if !output.ExpiresAt.Equal(expiresAt) {
			t.Errorf("ExpiresAt = %v、期待値 = %v", output.ExpiresAt, expiresAt)
		}
	})

	notFoundTests := []struct {
		name       string
		principal  *model.APIPrincipal
		identifier model.SpaceIdentifier
	}{
		{name: "トークンが属さないスペースは未存在として扱う", principal: principal, identifier: "other-space"},
		{name: "主体が無い場合は未存在として扱う", principal: nil, identifier: "bound-space"},
	}
	for _, tt := range notFoundTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := NewGetAPICurrentTokenUsecase().Execute(context.Background(), GetAPICurrentTokenInput{
				Principal:       tt.principal,
				SpaceIdentifier: tt.identifier,
			})
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			ae := model.AsAppError(err)
			if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
				t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
			}
		})
	}
}
