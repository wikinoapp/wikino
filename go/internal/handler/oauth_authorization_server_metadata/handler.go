// Package oauth_authorization_server_metadataは、OAuthの認可サーバーのメタデータ (RFC 8414) のハンドラーを提供する
package oauth_authorization_server_metadata

import (
	"github.com/wikinoapp/wikino/go/internal/config"
)

// Handlerは認可サーバーのメタデータのHTTPハンドラー
type Handler struct {
	cfg *config.Config
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

// Metadataは認可サーバーのメタデータ (RFC 8414 §2)。
// 署名したトークンを発行しないため、`jwks_uri` は持たない
type Metadata struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	RevocationEndpoint                         string   `json:"revocation_endpoint"`
	ScopesSupported                            []string `json:"scopes_supported"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported"`
	GrantTypesSupported                        []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported"`
	RevocationEndpointAuthMethodsSupported     []string `json:"revocation_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	AuthorizationResponseIssParameterSupported bool     `json:"authorization_response_iss_parameter_supported"`
	ServiceDocumentation                       string   `json:"service_documentation"`
}
