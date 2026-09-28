package oauth_authorization_server_metadata

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// Showは認可サーバーのメタデータを返す (GET /.well-known/oauth-authorization-server)。
// 値は認可・トークン・失効のエンドポイントが要求の検証に使う定数と設定から組み立て、
// メタデータと実際の挙動がずれないようにする
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	appURL := h.cfg.AppURL()
	metadata := Metadata{
		// 認可レスポンスの `iss` (RFC 9207) と同じ値にする
		Issuer:                                 appURL,
		AuthorizationEndpoint:                  appURL + model.OAuthAuthorizationEndpointPath,
		TokenEndpoint:                          appURL + model.OAuthTokenEndpointPath,
		RevocationEndpoint:                     appURL + model.OAuthRevocationEndpointPath,
		ScopesSupported:                        model.ScopesToStrings(model.APITokenScopes),
		ResponseTypesSupported:                 []string{model.OAuthResponseTypeCode},
		ResponseModesSupported:                 []string{"query"},
		GrantTypesSupported:                    []string{model.OAuthGrantTypeAuthorizationCode, model.OAuthGrantTypeRefreshToken},
		TokenEndpointAuthMethodsSupported:      model.OAuthClientAuthMethods,
		RevocationEndpointAuthMethodsSupported: model.OAuthClientAuthMethods,
		CodeChallengeMethodsSupported:          []string{model.OAuthCodeChallengeMethodS256},
		// 認可レスポンスに `iss` を付ける (RFC 9207 §3)
		AuthorizationResponseIssParameterSupported: true,
		// 開発者向けのドキュメント。OAuthの流れとスコープはAPIリファレンスで説明する
		ServiceDocumentation: appURL + model.APIReferencePath,
	}

	header := w.Header()
	header.Set("Content-Type", "application/json")
	// 設定を変えるのはデプロイのときだけのため、クライアントや中継のキャッシュに1時間残してよい
	header.Set("Cache-Control", "public, max-age=3600")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		slog.ErrorContext(r.Context(), "認可サーバーのメタデータの書き込みに失敗", "error", err)
	}
}
