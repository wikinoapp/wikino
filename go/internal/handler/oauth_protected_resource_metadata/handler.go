// Package oauth_protected_resource_metadataは、スペースのAPIの保護リソースのメタデータ (RFC 9728) の
// ハンドラーを提供する
package oauth_protected_resource_metadata

import (
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは保護リソースのメタデータのHTTPハンドラー
type Handler struct {
	cfg                                 *config.Config
	getOAuthProtectedResourceMetadataUC *usecase.GetOAuthProtectedResourceMetadataUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(cfg *config.Config, getOAuthProtectedResourceMetadataUC *usecase.GetOAuthProtectedResourceMetadataUsecase) *Handler {
	return &Handler{
		cfg:                                 cfg,
		getOAuthProtectedResourceMetadataUC: getOAuthProtectedResourceMetadataUC,
	}
}

// Metadataは保護リソースのメタデータ (RFC 9728 §2)。
// 署名したトークンを受け付けないため、`jwks_uri` は持たない
type Metadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
	ResourceDocumentation  string   `json:"resource_documentation"`
}
