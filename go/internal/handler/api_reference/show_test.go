package api_reference_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/api_reference"
)

func TestShow(t *testing.T) {
	t.Parallel()

	handler := api_reference.NewHandler(&config.Config{Env: "test", Domain: "example.com", GitRev: "abc123"})
	req := httptest.NewRequest(http.MethodGet, "/api/reference/v1", nil)
	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = 200", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("Content-Type = %q、HTMLを期待", got)
	}
	// 外部リソースと埋め込みを制限し、Redocに必要なスタイルとWorkerだけ許可する
	if got, want := rr.Header().Get("Content-Security-Policy"), "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; worker-src 'self' blob:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"; got != want {
		t.Errorf("Content-Security-Policy = %q、期待値 = %q", got, want)
	}

	body := rr.Body.String()
	wantContains := []string{
		"<title>Wikino API V1リファレンス</title>",
		`<link rel="canonical" href="https://example.com/api/reference/v1">`,
		// 描画するOpenAPI記述と、機械可読な記述への案内 (RFC 8631)
		`data-spec-url="/api/v1/openapi.yaml"`,
		`<link rel="service-desc" href="/api/v1/openapi.yaml" type="application/yaml">`,
		`<script type="module" src="/static/js/api-reference.js?v=abc123"></script>`,
		// JavaScriptの読み込みに失敗しても主要な情報とOpenAPI記述へ辿れる
		`<h1>Wikino API V1リファレンス</h1>`,
		`Wikinoの公開Web API V1のリファレンスです。`,
		`id="api-reference-fallback"`,
		`<a href="/api/v1/openapi.yaml">openapi.yaml</a>`,
	}
	for _, want := range wantContains {
		if !strings.Contains(body, want) {
			t.Errorf("本文に %q が含まれない", want)
		}
	}
	// Redocの表示に干渉するWikinoのCSSとスクリプトは読み込まない
	for _, notWant := range []string{"/static/css/style.css", "/static/js/main.js"} {
		if strings.Contains(body, notWant) {
			t.Errorf("本文に %q が含まれる", notWant)
		}
	}
}
