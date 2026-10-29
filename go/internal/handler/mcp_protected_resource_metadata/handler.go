// Package mcp_protected_resource_metadataは、リモートMCPサーバーの保護リソースのメタデータ (RFC 9728) の
// ハンドラーを提供する
package mcp_protected_resource_metadata

import (
	"github.com/wikinoapp/wikino/go/internal/config"
)

// Handlerは、リモートMCPサーバーの保護リソースのメタデータのHTTPハンドラー
type Handler struct {
	cfg *config.Config
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

// Metadataは保護リソースのメタデータ (RFC 9728 §2)。
// 署名したトークンを受け付けないため、`jwks_uri` は持たない
type Metadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}
