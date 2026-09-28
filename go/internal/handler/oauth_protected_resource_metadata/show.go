package oauth_protected_resource_metadata

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// ShowはスペースのAPIの保護リソースのメタデータを返す
// (GET /.well-known/oauth-protected-resource/api/v1/spaces/{space_identifier})。
// 保護リソースの識別子は、認可要求の `resource` (RFC 8707) と同じスペースのAPIのURLにする
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	output, err := h.getOAuthProtectedResourceMetadataUC.Execute(ctx, usecase.GetOAuthProtectedResourceMetadataInput{
		SpaceIdentifier: model.SpaceIdentifier(chi.URLParam(r, "space_identifier")),
	})
	if err != nil {
		if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeResourceNotFound {
			w.Header().Set("Cache-Control", "no-store")
			httperror.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "保護リソースのメタデータのデータの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	appURL := h.cfg.AppURL()
	metadata := Metadata{
		Resource: model.SpaceAPIResourceURL(appURL, output.Space.Identifier),
		// 認可サーバーのメタデータの `issuer` と同じ値にする
		AuthorizationServers: []string{appURL},
		ScopesSupported:      model.ScopesToStrings(model.APITokenScopes),
		// アクセストークンはクエリパラメーターや本文では受け付けず、`Authorization` ヘッダーだけで受け付ける
		BearerMethodsSupported: []string{"header"},
		// スペースごとに内容は変わらないため、APIリファレンスを指す
		ResourceDocumentation: appURL + model.APIReferencePath,
	}

	header := w.Header()
	header.Set("Content-Type", "application/json")
	// 設定を変えるのはデプロイのときだけのため、クライアントや中継のキャッシュに1時間残してよい
	header.Set("Cache-Control", "public, max-age=3600")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		slog.ErrorContext(ctx, "保護リソースのメタデータの書き込みに失敗", "error", err)
	}
}
