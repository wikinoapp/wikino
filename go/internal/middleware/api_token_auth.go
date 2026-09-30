package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/model"
)

const (
	// apiPrincipalContextKeyはコンテキストに公開APIの呼び出し主体を保存するためのキー
	apiPrincipalContextKey contextKey = "api_principal"
)

// apiTokenAuthenticatorは公開APIのトークンを照合するインターフェース。
// usecase.AuthenticateAPITokenUsecaseがこのインターフェースを満たす。
// トークンを受け付けられない場合は (nil, nil) を返す
type apiTokenAuthenticator interface {
	Execute(ctx context.Context, token string) (*model.APIPrincipal, error)
}

// APITokenAuthは `Authorization: Bearer` のトークンを照合し、呼び出し主体をコンテキストに入れるミドルウェア。
// トークンの無いリクエストはそのまま通し、トークンが要るかどうかはAPIRequestValidatorが
// operationのsecurityで判定する
type APITokenAuth struct {
	authenticator apiTokenAuthenticator
	problems      *apierror.Writer
}

// NewAPITokenAuthは新しいAPITokenAuthを作成する
func NewAPITokenAuth(authenticator apiTokenAuthenticator, problems *apierror.Writer) *APITokenAuth {
	return &APITokenAuth{
		authenticator: authenticator,
		problems:      problems,
	}
}

// MiddlewareはHTTPミドルウェアを返す
func (a *APITokenAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}

		// 送られてきたトークンを受け付けられない場合は、operationがトークンを要るかどうかに
		// かかわらず401にする。無効なトークンを黙って無視すると、クライアントが設定の誤りに気付けないため
		token, ok := bearerToken(header)
		if !ok {
			a.problems.Unauthorized(w, r, apierror.BearerChallenge{Error: "invalid_token"})
			return
		}

		principal, err := a.authenticator.Execute(r.Context(), token)
		if err != nil {
			slog.ErrorContext(r.Context(), "APIのトークンの照合に失敗しました", "error", err)
			a.problems.InternalServerError(w, r)
			return
		}
		if principal == nil {
			a.problems.Unauthorized(w, r, apierror.BearerChallenge{Error: "invalid_token"})
			return
		}

		next.ServeHTTP(w, r.WithContext(SetAPIPrincipalToContext(r.Context(), principal)))
	})
}

// bearerTokenは `Authorization` ヘッダーの値からBearerトークンを取り出す。
// 認証スキームの名前は大文字小文字を区別しない (RFC 9110 §11.1)
func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	return token, true
}

// APIPrincipalFromContextはコンテキストから公開APIの呼び出し主体を取得する。
// トークンの無いリクエストではnilを返す
func APIPrincipalFromContext(ctx context.Context) *model.APIPrincipal {
	principal, ok := ctx.Value(apiPrincipalContextKey).(*model.APIPrincipal)
	if !ok {
		return nil
	}
	return principal
}

// SetAPIPrincipalToContextはコンテキストに公開APIの呼び出し主体を設定する。
// テストでトークン認証済みの状態を再現するためにも使う
func SetAPIPrincipalToContext(ctx context.Context, principal *model.APIPrincipal) context.Context {
	return context.WithValue(ctx, apiPrincipalContextKey, principal)
}
