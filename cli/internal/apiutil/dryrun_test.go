package apiutil

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

func TestDryRun(t *testing.T) {
	t.Parallel()

	var sent []string
	dryRun := &DryRun{}
	c := newTestClient(t, i18n.NewLocalizer(i18n.LangJa), dryRun, func(req *http.Request) (*http.Response, error) {
		sent = append(sent, req.Method+" "+req.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"links":[]}`)),
		}, nil
	})
	ctx := context.Background()

	// 読み取り専用の要求 (Wikiリンクのプレビュー) は `--dry-run` でも送る。
	if _, err := c.CreateWikilinkPreviewWithResponse(ctx, "my-space", apigen.NewWikilinkPreview{TopicNumber: 3, Body: "[[a]]"}); err != nil {
		t.Fatalf("読み取りの要求に失敗しました: %v", err)
	}

	etag := `"v1"`
	title := "新しいタイトル"
	_, err := c.UpdatePageWithApplicationMergePatchPlusJSONBodyWithResponse(ctx, "my-space", 12,
		&apigen.UpdatePageParams{IfMatch: &etag}, apigen.PagePatch{Title: &title})
	if !errors.Is(err, ErrDryRun) {
		t.Fatalf("ErrDryRunが返りません: %v", err)
	}

	if len(sent) != 1 || sent[0] != "POST /api/v1/spaces/my-space/wikilink_previews" {
		t.Errorf("送った要求が違います: %v", sent)
	}

	requests := dryRun.Requests()
	if len(requests) != 1 {
		t.Fatalf("記録した要求の数が違います: %d", len(requests))
	}
	r := requests[0]
	if r.Method != http.MethodPatch || r.URL != testHost+"/api/v1/spaces/my-space/pages/12" {
		t.Errorf("記録したメソッドとURLが違います: %s %s", r.Method, r.URL)
	}
	if r.Headers["Authorization"] != "Bearer [REDACTED]" {
		t.Errorf("Authorizationが伏せられていません: %q", r.Headers["Authorization"])
	}
	if r.Headers["If-Match"] != `"v1"` {
		t.Errorf("If-Matchが記録されていません: %v", r.Headers)
	}

	encoded, err := json.Marshal(requests)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), testToken) {
		t.Errorf("記録した要求にトークンが含まれています: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"body":{"title":"新しいタイトル"}`) {
		t.Errorf("本文がJSONとして記録されていません: %s", encoded)
	}
}

func TestDryRun_IncludeReadOnly(t *testing.T) {
	t.Parallel()

	dryRun := &DryRun{IncludeReadOnly: true}
	c := newTestClient(t, i18n.NewLocalizer(i18n.LangJa), dryRun, func(*http.Request) (*http.Response, error) {
		t.Error("IncludeReadOnlyのdry-runで要求が送信されました")
		return nil, errors.New("送信されました")
	})

	// 読み取り専用の要求も送らずに記録する。
	_, err := c.Do(context.Background(), http.MethodGet, "/spaces/my-space/pages", nil, nil)
	if !errors.Is(err, ErrDryRun) {
		t.Fatalf("ErrDryRunが返りません: %v", err)
	}

	out := dryRun.Output()
	if !out.DryRun || len(out.Requests) != 1 {
		t.Fatalf("dry-runの出力が違います: %+v", out)
	}
	r := out.Requests[0]
	if r.Method != http.MethodGet || r.URL != testHost+"/api/v1/spaces/my-space/pages" || r.Headers["Authorization"] != "Bearer "+redacted {
		t.Errorf("記録した要求が違います: %+v", r)
	}
}

func TestRedactCredentials(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"Bearer abc":  "Bearer [REDACTED]",
		"Basic xyz==": "Basic [REDACTED]",
		"rawsecret":   "[REDACTED]",
	}
	for in, want := range tests {
		if got := redactCredentials(in); got != want {
			t.Errorf("%q: 実際 %q, 期待 %q", in, got, want)
		}
	}
}

func TestDryRun_SecretHeaders(t *testing.T) {
	t.Parallel()
	dryRun := &DryRun{}
	c := newTestClient(t, i18n.NewLocalizer(i18n.LangJa), dryRun, func(*http.Request) (*http.Response, error) {
		t.Error("dry-runの書き込みが送信されました")
		return nil, errors.New("送信されました")
	})
	_, err := c.CreatePageWithResponse(context.Background(), "my-space", apigen.NewPage{TopicNumber: 1, Title: "ページ", Body: "本文"}, func(_ context.Context, req *http.Request) error {
		req.Header.Add("Cookie", "session=secret-session; preference=secret-preference")
		req.Header.Add("Cookie", "another=secret-another")
		req.Header.Set("Proxy-Authorization", "Basic secret-proxy")
		return nil
	})
	if !errors.Is(err, ErrDryRun) {
		t.Fatalf("dry-runで要求を止められません: %v", err)
	}
	requests := dryRun.Requests()
	if len(requests) != 1 {
		t.Fatalf("記録した要求の数が違います: %d", len(requests))
	}
	headers := requests[0].Headers
	if headers["Cookie"] != redacted || headers["Authorization"] != "Bearer "+redacted || headers["Proxy-Authorization"] != "Basic "+redacted {
		t.Errorf("秘密のヘッダーが伏せられていません: %v", headers)
	}
	encoded, err := json.Marshal(requests)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{testToken, "secret-session", "secret-preference", "secret-another", "secret-proxy"} {
		if strings.Contains(string(encoded), secret) {
			t.Errorf("要求のJSONに秘密が残っています: %s", encoded)
		}
	}
}
