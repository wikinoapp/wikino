package draft

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// testRevisionは、APIの下書きのリビジョンの応答の1件を返す。
func testRevision(n int, title *string, body string) map[string]any {
	return map[string]any{
		"id":         fmt.Sprintf("00000000-0000-0000-0000-%012d", n),
		"title":      title,
		"body":       body,
		"created_at": "2026-10-01T09:00:00.123456Z",
	}
}

// handleRevisionPagesは、ページ12の下書きの編集履歴の一覧の要求に、カーソルごとのページで応答する。
// pagesのキーはカーソル (最初のページは空文字列)、値はそのページのリビジョンと次のカーソル。
func handleRevisionPages(t *testing.T, e *cmdtest.Env, pages map[string]revisionPage) {
	t.Helper()

	e.Handle(http.MethodGet, revisionsPath, func(req *http.Request) *http.Response {
		page, ok := pages[req.URL.Query().Get("cursor")]
		if !ok {
			t.Errorf("想定外のカーソル: %q", req.URL.Query().Get("cursor"))
		}
		return cmdtest.JSON(http.StatusOK, map[string]any{"items": page.items, "next_cursor": page.next})
	})
}

type revisionPage struct {
	items []map[string]any
	next  *string
}

func TestRevisionList_JSON(t *testing.T) {
	t.Parallel()

	longBody := "# 議事録\n\n" + strings.Repeat("あいうえお", 20)
	e := newEnv(t)
	handleRevisionPages(t, e, map[string]revisionPage{
		"": {items: []map[string]any{testRevision(1, ptr("議事録"), longBody), testRevision(2, nil, "")}, next: ptr("cursor-2")},
	})

	if err := e.Run(NewCmdDraft, "draft", "revision", "list", "12"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	var got listOutput
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if len(got.Items) != 2 {
		t.Fatalf("件数 = %d、期待値 = 2", len(got.Items))
	}
	first := got.Items[0]
	if _, ok := first["body"]; ok {
		t.Errorf("一覧の既定の出力に本文が含まれる: %v", first)
	}
	if excerpt, _ := first["excerpt"].(string); !strings.HasPrefix(excerpt, "# 議事録 あいうえお") || !strings.HasSuffix(excerpt, "…") {
		t.Errorf("excerpt = %q", first["excerpt"])
	}
	if first["id"] != "00000000-0000-0000-0000-000000000001" || first["title"] != "議事録" || first["created_at"] != "2026-10-01T09:00:00.123456Z" {
		t.Errorf("1件目 = %v", first)
	}
	if second := got.Items[1]; second["title"] != nil || second["excerpt"] != "" {
		t.Errorf("2件目 = %v", second)
	}
	if got.NextCursor == nil || *got.NextCursor != "cursor-2" {
		t.Errorf("next_cursor = %v、期待値 = cursor-2", got.NextCursor)
	}
	if !strings.Contains(e.Stderr.String(), "--cursor cursor-2") {
		t.Errorf("標準エラーに次のページの取り方が無い: %q", e.Stderr.String())
	}
}

func TestRevisionList_Paging(t *testing.T) {
	t.Parallel()

	pages := map[string]revisionPage{
		"":         {items: []map[string]any{testRevision(1, ptr("議事録"), "1\n")}, next: ptr("cursor-2")},
		"cursor-2": {items: []map[string]any{testRevision(2, ptr("議事録"), "2\n")}},
	}

	t.Run("--cursor", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		handleRevisionPages(t, e, pages)
		if err := e.Run(NewCmdDraft, "draft", "revision", "list", "12", "--cursor", "cursor-2", "--limit", "1"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}

		var got listOutput
		cmdtest.Decode(t, e.Stdout.String(), &got)
		if len(got.Items) != 1 || got.Items[0]["id"] != "00000000-0000-0000-0000-000000000002" || got.NextCursor != nil {
			t.Errorf("出力 = %s", e.Stdout.String())
		}
		if q := e.Requests()[0].URL.Query(); q.Get("limit") != "1" || q.Get("cursor") != "cursor-2" {
			t.Errorf("クエリ = %v", q)
		}
	})

	t.Run("--all", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		handleRevisionPages(t, e, pages)
		if err := e.Run(NewCmdDraft, "draft", "revision", "list", "12", "--all", "--fields", "id,body"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}

		// `next_cursor` を辿って全件を1行1件で出す。
		lines := cmdtest.Lines(e.Stdout.String())
		want := []string{
			`{"body":"1\n","id":"00000000-0000-0000-0000-000000000001"}`,
			`{"body":"2\n","id":"00000000-0000-0000-0000-000000000002"}`,
		}
		if strings.Join(lines, "\n") != strings.Join(want, "\n") {
			t.Errorf("出力 = %q、期待値 = %q", lines, want)
		}
	})
}

func TestRevisionList_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handleRevisionPages(t, e, map[string]revisionPage{
		"": {items: []map[string]any{testRevision(1, ptr("議事録"), "本文\n"), testRevision(2, nil, "")}},
	})

	if err := e.RunTTY(NewCmdDraft, "draft", "revision", "list", "12"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "revision_list", e.Stdout.String())
}

func TestRevisionList_Empty(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handleRevisionPages(t, e, map[string]revisionPage{"": {items: []map[string]any{}}})

	if err := e.RunTTY(NewCmdDraft, "draft", "revision", "list", "12"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if e.Stdout.Len() > 0 {
		t.Errorf("標準出力 = %q、期待値 = 空", e.Stdout.String())
	}
	if !strings.Contains(e.Stderr.String(), "自動保存ではリビジョンを作りません") {
		t.Errorf("標準エラーにヒントが無い: %q", e.Stderr.String())
	}
}

func TestRevisionList_NotFound(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, revisionsPath, func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "下書きが見つかりません。")
	})

	err := e.Run(NewCmdDraft, "draft", "revision", "list", "12")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "wikino draft list") {
		t.Errorf("ヒントに下書きの一覧のコマンドが無い: %q", cliErr.Hint)
	}
}

func TestRevisionList_UsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{name: "ページ番号が無い", args: []string{}, wantMessage: "1"},
		{name: "不正なページ番号", args: []string{"0"}, wantMessage: "<page_number>"},
		{name: "--allと--cursorの併用", args: []string{"12", "--all", "--cursor", "x"}, wantMessage: "--all, --cursor"},
		{name: "未知の項目", args: []string{"12", "--fields", "etag"}, wantMessage: "etag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdDraft, append([]string{"draft", "revision", "list"}, tt.args...)...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, tt.wantMessage) {
				t.Errorf("メッセージ %q に %q が無い", cliErr.Message, tt.wantMessage)
			}
			assertNoRequest(t, e)
		})
	}
}

func TestRevision_SubcommandRequired(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	err := e.Run(NewCmdDraft, "draft", "revision")
	_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
	assertNoRequest(t, e)
}
