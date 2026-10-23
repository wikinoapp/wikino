package apiutil

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestNewClient_Headers(t *testing.T) {
	t.Parallel()

	var got *http.Request
	c := newTestClient(t, i18n.NewLocalizer(i18n.LangJa), nil, func(req *http.Request) (*http.Response, error) {
		got = req
		return problemResponse(http.StatusNotFound, `{"type":"about:blank","title":"Not Found","status":404,"detail":null}`), nil
	})

	_, _ = c.GetPageWithResponse(context.Background(), "my-space", 12, nil)

	if got == nil {
		t.Fatal("要求が送られていません")
	}
	if ua := got.Header.Get("User-Agent"); !strings.HasPrefix(ua, "wikino-cli/1.2.3 (") {
		t.Errorf("User-Agentが違います: %q", ua)
	}
	if lang := got.Header.Get("Accept-Language"); lang != "ja" {
		t.Errorf("Accept-Languageが表示言語と違います: %q", lang)
	}
}

func TestNewClient_InvalidHost(t *testing.T) {
	t.Parallel()

	_, err := NewClient(i18n.NewLocalizer(i18n.LangJa), ClientOptions{Host: "http://example.com"})

	var cliErr *output.Error
	if !errors.As(err, &cliErr) {
		t.Fatalf("CLIのエラーが返りません: %v", err)
	}
	if cliErr.ExitCode != output.ExitUsage || !strings.Contains(cliErr.Message, "HTTPS") {
		t.Errorf("接続先の誤りのエラーが違います: %+v", cliErr)
	}
}

func TestUserAgent(t *testing.T) {
	t.Parallel()

	ua := UserAgent("1.0.0")
	if !strings.HasPrefix(ua, "wikino-cli/1.0.0 (") || !strings.HasSuffix(ua, ")") {
		t.Errorf("User-Agentの形が違います: %q", ua)
	}
}
