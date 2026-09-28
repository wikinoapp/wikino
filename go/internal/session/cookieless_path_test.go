package session_test

import (
	"net/http"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/session"
)

func TestIsCookielessRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{name: "添付ファイルのog:image", method: http.MethodGet, path: "/attachments/550e8400-e29b-41d4-a716-446655440000/og_image", want: true},
		{name: "添付ファイルのog:imageのHEAD", method: http.MethodHead, path: "/attachments/550e8400-e29b-41d4-a716-446655440000/og_image", want: true},
		{name: "添付ファイルのog:imageへのPOST", method: http.MethodPost, path: "/attachments/550e8400-e29b-41d4-a716-446655440000/og_image", want: false},
		{name: "ページのカード画像", method: http.MethodGet, path: "/s/example/pages/1/og_image/0123456789abcdef.png", want: true},
		{name: "形式の違うバージョンのカード画像", method: http.MethodGet, path: "/s/example/pages/1/og_image/old.png", want: true},
		{name: "添付ファイルのダウンロードURL", method: http.MethodGet, path: "/attachments/550e8400-e29b-41d4-a716-446655440000", want: false},
		{name: "og_imageの後ろにパスが続く添付ファイル", method: http.MethodGet, path: "/attachments/550e8400-e29b-41d4-a716-446655440000/og_image/extra", want: false},
		{name: "拡張子がpngでないカード画像", method: http.MethodGet, path: "/s/example/pages/1/og_image/0123456789abcdef.jpg", want: false},
		{name: "バージョンの無いカード画像", method: http.MethodGet, path: "/s/example/pages/1/og_image", want: false},
		{name: "ページ表示画面", method: http.MethodGet, path: "/s/example/pages/1", want: false},
		{name: "APIのGET", method: http.MethodGet, path: "/api/v1/openapi.yaml", want: true},
		{name: "APIのPOST", method: http.MethodPost, path: "/api/v1/spaces/example/pages", want: true},
		{name: "APIのPATCH", method: http.MethodPatch, path: "/api/v1/spaces/example/pages/1", want: true},
		{name: "APIリファレンス", method: http.MethodGet, path: "/api/reference/v1", want: true},
		{name: "OAuthのトークン発行", method: http.MethodPost, path: "/oauth/token", want: true},
		{name: "OAuthのトークン失効", method: http.MethodPost, path: "/oauth/revoke", want: true},
		{name: "OAuthの同意画面", method: http.MethodGet, path: "/oauth/authorize", want: false},
		{name: "OAuthのトークン発行に似たパス", method: http.MethodPost, path: "/oauth/token/extra", want: false},
		{name: "認可サーバーのメタデータ", method: http.MethodGet, path: "/.well-known/oauth-authorization-server", want: true},
		{name: "保護リソースのメタデータ", method: http.MethodGet, path: "/.well-known/oauth-protected-resource/api/v1/spaces/example", want: true},
		{name: "APIカタログ", method: http.MethodGet, path: "/.well-known/api-catalog", want: true},
		{name: "対象外のwell-known", method: http.MethodGet, path: "/.well-known/security.txt", want: false},
		{name: "未知のバージョンのAPI", method: http.MethodGet, path: "/api/v2/user", want: true},
		{name: "末尾のスラッシュが無い/api", method: http.MethodGet, path: "/api", want: false},
		{name: "apiで始まる別のパス", method: http.MethodGet, path: "/apis", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := session.IsCookielessRequest(tt.method, tt.path); got != tt.want {
				t.Errorf("IsCookielessRequest(%q, %q) = %v、期待値 = %v", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestIsAPIPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{path: "/api/v1/openapi.yaml", want: true},
		{path: "/api/", want: true},
		{path: "/api/reference/v1", want: false},
		{path: "/api/reference/v1/", want: true},
		{path: "/api", want: false},
		{path: "/apis/v1", want: false},
		{path: "/s/example/api/v1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			if got := session.IsAPIPath(tt.path); got != tt.want {
				t.Errorf("IsAPIPath(%q) = %v、期待値 = %v", tt.path, got, tt.want)
			}
		})
	}
}
