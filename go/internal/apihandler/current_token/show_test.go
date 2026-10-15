package current_token_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/current_token"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestHandler_GetCurrentToken(t *testing.T) {
	t.Parallel()

	h := current_token.NewHandler(usecase.NewGetAPICurrentTokenUsecase())
	newContext := func(kind model.APITokenKind, scopes []model.Scope, expiresAt time.Time) context.Context {
		return middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
			User:           &model.User{ID: "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607182"},
			Space:          &model.Space{ID: "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607183", Identifier: "alice-wiki"},
			SpaceMember:    &model.SpaceMember{ID: "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607184", SpaceID: "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607183", UserID: "0198f0a6-7c3e-7d1a-9b2c-3d4e5f607182"},
			TokenKind:      kind,
			TokenExpiresAt: expiresAt,
			Scopes:         scopes,
		})
	}

	tests := []struct {
		name       string
		kind       model.APITokenKind
		scopes     []model.Scope
		expiresAt  time.Time
		wantKind   apigen.TokenKind
		wantScopes []string
	}{
		{
			name:       "個人アクセストークンの種類・スコープ・有効期限を返す",
			kind:       model.APITokenKindPersonalAccessToken,
			scopes:     []model.Scope{model.ScopePageWrite, model.ScopePageRead},
			expiresAt:  time.Date(2026, 12, 31, 15, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
			wantKind:   apigen.PersonalAccessToken,
			wantScopes: []string{"page:read", "page:write"},
		},
		{
			name:       "OAuthのアクセストークンの種類・スコープ・有効期限を返す",
			kind:       model.APITokenKindOAuthAccessToken,
			scopes:     []model.Scope{model.ScopeDraftPageRead},
			expiresAt:  time.Date(2026, 10, 13, 1, 2, 3, 456000, time.UTC),
			wantKind:   apigen.OauthAccessToken,
			wantScopes: []string{"draft_page:read"},
		},
		{
			name:       "スコープの無いトークンは空の配列を返す",
			kind:       model.APITokenKindPersonalAccessToken,
			scopes:     nil,
			expiresAt:  time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			wantKind:   apigen.PersonalAccessToken,
			wantScopes: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := h.GetCurrentToken(newContext(tt.kind, tt.scopes, tt.expiresAt), apigen.GetCurrentTokenRequestObject{SpaceIdentifier: "alice-wiki"})
			if err != nil {
				t.Fatalf("GetCurrentToken() error = %v", err)
			}
			jsonRes, ok := res.(apigen.GetCurrentToken200JSONResponse)
			if !ok {
				t.Fatalf("レスポンスの型 = %T、期待値 = GetCurrentToken200JSONResponse", res)
			}

			if jsonRes.Body.Kind != tt.wantKind {
				t.Errorf("kind = %q、期待値 = %q", jsonRes.Body.Kind, tt.wantKind)
			}
			if !slices.Equal(jsonRes.Body.Scopes, tt.wantScopes) {
				t.Errorf("scopes = %v、期待値 = %v", jsonRes.Body.Scopes, tt.wantScopes)
			}
			if !jsonRes.Body.ExpiresAt.Equal(tt.expiresAt) || jsonRes.Body.ExpiresAt.Location() != time.UTC {
				t.Errorf("expires_at = %v、期待値 = %v (UTC)", jsonRes.Body.ExpiresAt, tt.expiresAt.UTC())
			}

			// トークンの値・ダイジェスト・名前など、種類・スコープ・有効期限の他の項目を返さない
			raw, err := json.Marshal(jsonRes.Body)
			if err != nil {
				t.Fatalf("本文のJSONへの変換に失敗: %v", err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatalf("本文のJSONを読めない: %v", err)
			}
			if len(fields) != 3 || fields["kind"] == nil || fields["scopes"] == nil || fields["expires_at"] == nil {
				t.Errorf("本文 = %s、期待値 = kind・scopes・expires_atだけを含む", raw)
			}
		})
	}

	t.Run("トークンが属さないスペースは未存在のエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctx := newContext(model.APITokenKindPersonalAccessToken, []model.Scope{model.ScopePageRead}, time.Now().Add(time.Hour))
		res, err := h.GetCurrentToken(ctx, apigen.GetCurrentTokenRequestObject{SpaceIdentifier: "other-wiki"})
		if res != nil {
			t.Errorf("レスポンス = %v、期待値 = nil", res)
		}
		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}
