// Package api_catalogは、公開Web APIのAPIカタログ (RFC 9727) のハンドラーを提供する
package api_catalog

import (
	"github.com/wikinoapp/wikino/go/internal/config"
)

// HandlerはAPIカタログのHTTPハンドラー
type Handler struct {
	cfg *config.Config
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

// LinksetはAPIカタログの文書 (RFC 9264のLinkset)
type Linkset struct {
	Linkset []LinkContext `json:"linkset"`
}

// LinkContextは1つのAPI (anchor) と、そのAPIを説明する文書へのリンク
type LinkContext struct {
	Anchor string `json:"anchor"`
	// ServiceDescは機械可読なAPIの記述 (RFC 8631)
	ServiceDesc []Link `json:"service-desc"`
	// ServiceDocは人間向けのAPIのドキュメント (RFC 8631)
	ServiceDoc []Link `json:"service-doc"`
}

// Linkはリンク先
type Link struct {
	Href string `json:"href"`
	Type string `json:"type"`
}
