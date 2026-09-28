package api_catalog_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/api_catalog"
)

func TestShow(t *testing.T) {
	t.Parallel()

	handler := api_catalog.NewHandler(&config.Config{Domain: "example.com"})
	req := httptest.NewRequest(http.MethodGet, "/.well-known/api-catalog", nil)
	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = 200", rr.Code)
	}
	wantHeader := map[string]string{
		"Content-Type":  `application/linkset+json; profile="https://www.rfc-editor.org/info/rfc9727"`,
		"Link":          `<https://example.com/.well-known/api-catalog>; rel="api-catalog"`,
		"Cache-Control": "public, max-age=3600",
	}
	for key, want := range wantHeader {
		if got := rr.Header().Get(key); got != want {
			t.Errorf("%s = %q、期待値 = %q", key, got, want)
		}
	}

	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("本文をJSONとして読めない: %v", err)
	}
	want := map[string]any{
		"linkset": []any{
			map[string]any{
				"anchor": "https://example.com/api/v1",
				"service-desc": []any{
					map[string]any{"href": "https://example.com/api/v1/openapi.yaml", "type": "application/yaml"},
				},
				"service-doc": []any{
					map[string]any{"href": "https://example.com/api/reference/v1", "type": "text/html"},
				},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("本文 = %v、期待値 = %v", got, want)
	}
}

// HEADでもカタログの場所を `Link` で示し、本文は送らない (RFC 9727 §2)。
// 本文を省くのはnet/httpのサーバーのため、レコーダーではなくテストサーバーを通して確かめる
func TestShow_HEAD(t *testing.T) {
	t.Parallel()

	handler := api_catalog.NewHandler(&config.Config{Domain: "example.com"})
	srv := httptest.NewServer(http.HandlerFunc(handler.Show))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodHead, srv.URL+"/.well-known/api-catalog", nil)
	if err != nil {
		t.Fatalf("リクエストの作成に失敗: %v", err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("HEADに失敗: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = 200", resp.StatusCode)
	}
	wantHeader := map[string]string{
		"Content-Type": `application/linkset+json; profile="https://www.rfc-editor.org/info/rfc9727"`,
		"Link":         `<https://example.com/.well-known/api-catalog>; rel="api-catalog"`,
	}
	for key, want := range wantHeader {
		if got := resp.Header.Get(key); got != want {
			t.Errorf("%s = %q、期待値 = %q", key, got, want)
		}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("本文の読み込みに失敗: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("本文 = %q、HEADでは空を期待", body)
	}
}
