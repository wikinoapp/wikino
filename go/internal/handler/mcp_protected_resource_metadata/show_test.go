package mcp_protected_resource_metadata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/mcp_protected_resource_metadata"
)

func TestShow(t *testing.T) {
	t.Parallel()

	handler := mcp_protected_resource_metadata.NewHandler(&config.Config{Domain: "example.com"})
	rr := httptest.NewRecorder()
	handler.Show(rr, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q、期待値 = application/json", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "public, max-age=3600" {
		t.Errorf("Cache-Control = %q、期待値 = public, max-age=3600", got)
	}

	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("本文のJSONの解析に失敗: %v", err)
	}
	want := map[string]any{
		"resource":                 "https://example.com/mcp",
		"authorization_servers":    []any{"https://example.com"},
		"scopes_supported":         []any{"topic:read", "page:read", "page:write", "draft_page:read", "draft_page:write", "draft_page:delete", "attachment:read"},
		"bearer_methods_supported": []any{"header"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("本文 = %v、期待値 = %v", got, want)
	}
}
