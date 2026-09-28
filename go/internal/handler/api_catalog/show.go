package api_catalog

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// contentTypeはAPIカタログのContent-Type。profileでLinksetがAPIカタログであることを示す (RFC 9727 §4.2)
const contentType = `application/linkset+json; profile="https://www.rfc-editor.org/info/rfc9727"`

// ShowはAPIカタログを返す (GET・HEAD /.well-known/api-catalog)。
// HEADにも `Link` ヘッダーでカタログの場所を示す (RFC 9727 §2)。本文はnet/httpがHEADでは送らない
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	appURL := h.cfg.AppURL()
	catalog := Linkset{
		Linkset: []LinkContext{
			{
				Anchor: appURL + model.APIBasePath,
				ServiceDesc: []Link{
					{Href: appURL + model.APIOpenAPIDescriptionPath, Type: "application/yaml"},
				},
				ServiceDoc: []Link{
					{Href: appURL + model.APIReferencePath, Type: "text/html"},
				},
			},
		},
	}

	header := w.Header()
	header.Set("Content-Type", contentType)
	header.Set("Link", "<"+appURL+model.APICatalogPath+`>; rel="api-catalog"`)
	// APIの追加・廃止はデプロイのときだけのため、クライアントや中継のキャッシュに1時間残してよい
	header.Set("Cache-Control", "public, max-age=3600")
	if err := json.NewEncoder(w).Encode(catalog); err != nil {
		slog.ErrorContext(r.Context(), "APIカタログの書き込みに失敗", "error", err)
	}
}
