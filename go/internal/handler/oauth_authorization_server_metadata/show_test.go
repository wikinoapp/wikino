package oauth_authorization_server_metadata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/oauth_authorization_server_metadata"
)

func TestShow(t *testing.T) {
	t.Parallel()

	handler := oauth_authorization_server_metadata.NewHandler(&config.Config{Domain: "example.com"})
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = 200", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q、期待値 = application/json", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "public, max-age=3600" {
		t.Errorf("Cache-Control = %q、期待値 = public, max-age=3600", got)
	}

	// 公開の契約のため、キーと値を固定の期待値で確かめる
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("本文をJSONとして読めない: %v", err)
	}
	want := map[string]any{
		"issuer":                                         "https://example.com",
		"authorization_endpoint":                         "https://example.com/oauth/authorize",
		"token_endpoint":                                 "https://example.com/oauth/token",
		"revocation_endpoint":                            "https://example.com/oauth/revoke",
		"scopes_supported":                               []any{"topic:read", "page:read", "page:write"},
		"response_types_supported":                       []any{"code"},
		"response_modes_supported":                       []any{"query"},
		"grant_types_supported":                          []any{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported":          []any{"client_secret_basic", "client_secret_post", "none"},
		"revocation_endpoint_auth_methods_supported":     []any{"client_secret_basic", "client_secret_post", "none"},
		"code_challenge_methods_supported":               []any{"S256"},
		"authorization_response_iss_parameter_supported": true,
		"service_documentation":                          "https://example.com/api/reference/v1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("本文 = %v、期待値 = %v", got, want)
	}
}
