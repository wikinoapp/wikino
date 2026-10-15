package page

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
)

const (
	pagesPath   = "/api/v1/spaces/team/pages"
	previewPath = "/api/v1/spaces/team/wikilink_previews"
)

func ptr[T any](v T) *T {
	return &v
}

// testPageは、APIのページの応答の1件を返す。
func testPage(number int32, title *string, body string) map[string]any {
	return map[string]any{
		"id":           fmt.Sprintf("00000000-0000-0000-0000-%012d", number),
		"number":       number,
		"title":        title,
		"status":       "published",
		"topic_number": 3,
		"modified_at":  "2026-10-01T09:00:00.123456Z",
		"url":          fmt.Sprintf("https://wikino.example.com/s/team/pages/%d", number),
		"body":         body,
	}
}

func newEnv(t *testing.T) *cmdtest.Env {
	t.Helper()

	return cmdtest.New(t)
}

// assertNoRequestは、要求を送っていないことを確かめる。
func assertNoRequest(t *testing.T, e *cmdtest.Env) {
	t.Helper()

	if requests := e.Requests(); len(requests) > 0 {
		t.Errorf("要求を送らないことを期待したが %d 件送った: %s", len(requests), requests[0].URL)
	}
}

func listResponse(items []map[string]any, next *string) *http.Response {
	return cmdtest.JSON(http.StatusOK, map[string]any{"items": items, "next_cursor": next})
}

// dryRunOutputは、`--dry-run` の標準出力。
type dryRunOutput struct {
	DryRun   bool `json:"dry_run"`
	Requests []struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    map[string]any    `json:"body"`
	} `json:"requests"`
}

func decodeDryRun(t *testing.T, s string) dryRunOutput {
	t.Helper()

	var got dryRunOutput
	cmdtest.Decode(t, s, &got)
	if !got.DryRun {
		t.Errorf("dry_runがtrueでない: %s", s)
	}

	return got
}

// assertNoSecretは、標準出力と標準エラーにトークンが含まれないことを確かめる。
func assertNoSecret(t *testing.T, e *cmdtest.Env) {
	t.Helper()

	if strings.Contains(e.Stdout.String()+e.Stderr.String(), cmdtest.Token) {
		t.Errorf("出力にトークンが含まれる:\n%s\n%s", e.Stdout.String(), e.Stderr.String())
	}
}

// previewLinkは、Wikiリンクのプレビューの応答のリンクの1件を返す。リンク先はトピック「開発」のtitleのページ。
func previewLink(title, resolution string) map[string]any {
	link := map[string]any{"text": title, "topic_name": "開発", "page_title": title, "resolution": resolution, "page": nil}
	if resolution == "existing_page" {
		link["page"] = map[string]any{
			"id":           "00000000-0000-0000-0000-000000000020",
			"number":       20,
			"status":       "published",
			"title":        title,
			"topic_number": 3,
			"url":          "https://wikino.example.com/s/team/pages/20",
		}
	}

	return link
}

// previewRecorderは、Wikiリンクのプレビューの要求で送った本文を記録する。
type previewRecorder struct {
	mu       sync.Mutex
	requests []map[string]any
}

// bodiesは、プレビューの要求で送った本文 (`body`) を送った順に返す。
func (r *previewRecorder) bodies() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	bodies := make([]string, 0, len(r.requests))
	for _, req := range r.requests {
		body, _ := req["body"].(string)
		bodies = append(bodies, body)
	}

	return bodies
}

// handlePreviewは、Wikiリンクのプレビューの要求を受けて記録し、どの本文にもlinksを返す。
func handlePreview(t *testing.T, e *cmdtest.Env, links ...map[string]any) *previewRecorder {
	t.Helper()

	if links == nil {
		links = []map[string]any{}
	}
	recorder := &previewRecorder{}
	e.Handle(http.MethodPost, previewPath, func(req *http.Request) *http.Response {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("要求の本文を読めない: %v", err)
		}
		var body map[string]any
		cmdtest.Decode(t, string(data), &body)
		recorder.mu.Lock()
		recorder.requests = append(recorder.requests, body)
		recorder.mu.Unlock()
		return cmdtest.JSON(http.StatusOK, map[string]any{"links": links})
	})

	return recorder
}

// handleCurrentPageは、ページ12の取得の要求に、トピック3のページと版currentETagで応答する。
func handleCurrentPage(e *cmdtest.Env, currentETag string) {
	e.Handle(http.MethodGet, pagesPath+"/12", func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testPage(12, ptr("議事録"), "# 議事録\n"), "ETag", currentETag)
	})
}
