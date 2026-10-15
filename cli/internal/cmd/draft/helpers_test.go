package draft

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
	draftPagesPath = "/api/v1/spaces/team/draft_pages"
	pagesPath      = "/api/v1/spaces/team/pages"
	previewPath    = "/api/v1/spaces/team/wikilink_previews"
	etag           = `"d41d8cd9"`
)

func ptr[T any](v T) *T {
	return &v
}

// draftPathは、ページnumberの下書きのパスを返す。
func draftPath(number int32) string {
	return fmt.Sprintf("%s/%d/draft_page", pagesPath, number)
}

// testDraftは、APIの下書きの応答の1件を返す。
func testDraft(number int32, title *string, body string) map[string]any {
	return map[string]any{
		"page_number":  number,
		"title":        title,
		"topic_number": 3,
		"modified_at":  "2026-10-01T09:00:00.123456Z",
		"url":          fmt.Sprintf("https://wikino.example.com/s/team/pages/%d/edit", number),
		"body":         body,
	}
}

// testPageは、APIのページの応答の1件を返す。下書きを作る既存のページの取得に使う。
func testPage(number, topic int32) map[string]any {
	return map[string]any{
		"id":           fmt.Sprintf("00000000-0000-0000-0000-%012d", number),
		"number":       number,
		"title":        "議事録",
		"status":       "published",
		"topic_number": topic,
		"modified_at":  "2026-10-01T09:00:00.123456Z",
		"url":          fmt.Sprintf("https://wikino.example.com/s/team/pages/%d", number),
		"body":         "# 議事録\n",
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

// assertNoSecretは、標準出力と標準エラーにトークンが含まれないことを確かめる。
func assertNoSecret(t *testing.T, e *cmdtest.Env) {
	t.Helper()

	if strings.Contains(e.Stdout.String()+e.Stderr.String(), cmdtest.Token) {
		t.Errorf("出力にトークンが含まれる:\n%s\n%s", e.Stdout.String(), e.Stderr.String())
	}
}

// requestPathsは、送った要求の「メソッド パス」を送った順に返す。
func requestPaths(e *cmdtest.Env) []string {
	requests := e.Requests()
	paths := make([]string, 0, len(requests))
	for _, req := range requests {
		paths = append(paths, req.Method+" "+req.URL.Path)
	}

	return paths
}

// previewRecorderは、Wikiリンクのプレビューの要求で送った本文を記録する。
type previewRecorder struct {
	mu       sync.Mutex
	requests []map[string]any
}

// sentは、プレビューの要求で送った本文を送った順に返す。
func (r *previewRecorder) sent() []map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.requests
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

// newPageLinkは、Wikiリンクのプレビューの応答で、書き込むと作られるリンク先の1件を返す。
func newPageLink(title string) map[string]any {
	return map[string]any{"text": title, "topic_name": "開発", "page_title": title, "resolution": "new_page", "page": nil}
}
