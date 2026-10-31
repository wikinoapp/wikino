package mcp_protected_resource_metadata

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// ShowはリモートMCPサーバーの保護リソースのメタデータを返す (GET /.well-known/oauth-protected-resource/mcp)。
// MCPクライアントは、`/mcp` の401の `WWW-Authenticate` からこのメタデータを見つけ、認可サーバーへ進む。
// 保護リソースの識別子は、認可要求の `resource` (RFC 8707) と同じMCPサーバーのURLにする
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	appURL := h.cfg.AppURL()
	metadata := Metadata{
		Resource: model.MCPServerResourceURL(appURL),
		// 認可サーバーのメタデータの `issuer` と同じ値にする
		AuthorizationServers: []string{appURL},
		ScopesSupported:      model.ScopesToStrings(model.APITokenScopes),
		// アクセストークンはクエリパラメーターや本文では受け付けず、`Authorization` ヘッダーだけで受け付ける
		BearerMethodsSupported: []string{"header"},
	}

	header := w.Header()
	header.Set("Content-Type", "application/json")
	// 設定を変えるのはデプロイのときだけのため、クライアントや中継のキャッシュに1時間残してよい
	header.Set("Cache-Control", "public, max-age=3600")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		slog.ErrorContext(r.Context(), "MCPサーバーの保護リソースのメタデータの書き込みに失敗", "error", err)
	}
}
