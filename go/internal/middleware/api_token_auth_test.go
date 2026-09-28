package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// stubAPITokenAuthenticatorは、tokensに登録したトークンだけを受け付けるトークンの照合
type stubAPITokenAuthenticator struct {
	tokens map[string]*model.APIPrincipal
	err    error
}

func (s stubAPITokenAuthenticator) Execute(_ context.Context, token string) (*model.APIPrincipal, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.tokens[token], nil
}

func TestAPITokenAuth_Middleware(t *testing.T) {
	t.Parallel()

	principal := &model.APIPrincipal{
		User:      &model.User{ID: "user-1"},
		TokenKind: model.APITokenKindPersonalAccessToken,
		Scopes:    []model.Scope{model.ScopePageRead},
	}
	authenticator := stubAPITokenAuthenticator{tokens: map[string]*model.APIPrincipal{"wkp_valid": principal}}

	tests := []struct {
		name          string
		authenticator stubAPITokenAuthenticator
		authorization string
		// wantPrincipalは次のハンドラーがコンテキストから受け取る主体。wantStatusが200のときだけ確かめる
		wantPrincipal *model.APIPrincipal
		wantStatus    int
		wantChallenge string
	}{
		{
			name:          "トークンの無いリクエストは主体なしで通す",
			authenticator: authenticator,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "受け付けたトークンの主体をコンテキストに入れる",
			authenticator: authenticator,
			authorization: "Bearer wkp_valid",
			wantPrincipal: principal,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "認証スキームの名前は大文字小文字を区別しない",
			authenticator: authenticator,
			authorization: "bearer wkp_valid",
			wantPrincipal: principal,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "受け付けられないトークンは401",
			authenticator: authenticator,
			authorization: "Bearer wkp_unknown",
			wantStatus:    http.StatusUnauthorized,
			wantChallenge: `Bearer error="invalid_token"`,
		},
		{
			name:          "Bearer以外の認証スキームは401",
			authenticator: authenticator,
			authorization: "Basic dXNlcjpwYXNz",
			wantStatus:    http.StatusUnauthorized,
			wantChallenge: `Bearer error="invalid_token"`,
		},
		{
			name:          "トークンの空なBearerは401",
			authenticator: authenticator,
			authorization: "Bearer ",
			wantStatus:    http.StatusUnauthorized,
			wantChallenge: `Bearer error="invalid_token"`,
		},
		{
			name:          "照合に失敗したら500",
			authenticator: stubAPITokenAuthenticator{err: errors.New("DBに接続できない")},
			authorization: "Bearer wkp_valid",
			wantStatus:    http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotPrincipal *model.APIPrincipal
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPrincipal = middleware.APIPrincipalFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/pages", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rr := httptest.NewRecorder()
			middleware.NewAPITokenAuth(tt.authenticator, apierror.NewWriter("https://example.com")).Middleware(next).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("ステータス = %d、期待値 = %d (本文: %s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if got := rr.Header().Get("WWW-Authenticate"); got != tt.wantChallenge {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, tt.wantChallenge)
			}
			if tt.wantStatus != http.StatusOK {
				if got := rr.Header().Get("Content-Type"); got != apierror.ContentType {
					t.Errorf("Content-Type = %q、期待値 = %q", got, apierror.ContentType)
				}
				return
			}
			if gotPrincipal != tt.wantPrincipal {
				t.Errorf("コンテキストの主体 = %v、期待値 = %v", gotPrincipal, tt.wantPrincipal)
			}
		})
	}
}
