package apiutil

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

const (
	testHost  = "https://example.com"
	testToken = "wikino_pat_secret-token-value"
)

// roundTripFuncは関数をトランスポートにする。URLを書き換えずにトランスポートを差し替えて、
// 要求を検証し応答を返すために使う。
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func problemResponse(status int, body string, headers ...string) *http.Response {
	h := http.Header{"Content-Type": {"application/problem+json"}}
	for i := 0; i+1 < len(headers); i += 2 {
		h.Set(headers[i], headers[i+1])
	}

	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body))}
}

func newTestClient(t *testing.T, loc *i18n.Localizer, dryRun *DryRun, rt roundTripFunc) *apiclient.Client {
	t.Helper()

	c, err := NewClient(loc, ClientOptions{
		Host:       testHost,
		Token:      apiclient.StaticToken(testToken),
		Version:    "1.2.3",
		DryRun:     dryRun,
		HTTPClient: &http.Client{Transport: rt},
	})
	if err != nil {
		t.Fatalf("クライアントを作れません: %v", err)
	}

	return c
}
