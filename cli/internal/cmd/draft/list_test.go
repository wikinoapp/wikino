package draft

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

type listOutput struct {
	Items      []map[string]any `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

func listResponse(items []map[string]any, next *string) *http.Response {
	return cmdtest.JSON(http.StatusOK, map[string]any{"items": items, "next_cursor": next})
}

func TestList_JSON(t *testing.T) {
	t.Parallel()

	longBody := "# 議事録\n\n" + strings.Repeat("あいうえお", 20)
	e := newEnv(t)
	e.Handle(http.MethodGet, draftPagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{
			testDraft(12, ptr("議事録"), longBody),
			testDraft(30, nil, ""),
		}, ptr("cursor-2"))
	})

	if err := e.Run(NewCmdDraft, "draft", "list"); err != nil {
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
	if first["page_number"] != float64(12) || first["url"] != "https://wikino.example.com/s/team/pages/12/edit" {
		t.Errorf("1件目 = %v", first)
	}
	if first["topic_name"] != "トピック3" {
		t.Errorf("topic_name = %v、期待値 = トピック3", first["topic_name"])
	}
	for _, key := range []string{"title", "topic_number", "modified_at"} {
		if _, ok := first[key]; !ok {
			t.Errorf("1件目に %s が無い: %v", key, first)
		}
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

func TestList_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{
			testDraft(12, ptr("議事録"), "本文\n"),
			testDraft(30, nil, ""),
		}, nil)
	})

	if err := e.RunTTY(NewCmdDraft, "draft", "list"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "list", e.Stdout.String())
	if e.Stderr.Len() > 0 {
		t.Errorf("標準エラーに出力がある: %q", e.Stderr.String())
	}
}

func TestList_Params(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{}, nil)
	})

	if err := e.Run(NewCmdDraft, "draft", "list", "--limit", "50", "--cursor", "abc"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	requests := e.Requests()
	if len(requests) != 1 {
		t.Fatalf("要求の数 = %d、期待値 = 1", len(requests))
	}
	query := requests[0].URL.Query()
	if query.Get("limit") != "50" || query.Get("cursor") != "abc" {
		t.Errorf("クエリ = %v", query)
	}
}

func TestList_FieldsBody(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{testDraft(12, ptr("議事録"), "本文\n")}, nil)
	})

	if err := e.Run(NewCmdDraft, "draft", "list", "--fields", "page_number,body"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	want := `{"items":[{"body":"本文\n","page_number":12}],"next_cursor":null}` + "\n"
	if got := e.Stdout.String(); got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestList_FieldsTopicName(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{testDraft(12, ptr("議事録"), "本文\n")}, nil)
	})

	if err := e.Run(NewCmdDraft, "draft", "list", "--fields", "page_number,topic_name"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	want := `{"items":[{"page_number":12,"topic_name":"トピック3"}],"next_cursor":null}` + "\n"
	if got := e.Stdout.String(); got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestList_All(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPagesPath, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("cursor") == "" {
			return listResponse([]map[string]any{testDraft(12, ptr("議事録"), "本文\n")}, ptr("cursor-2"))
		}
		return listResponse([]map[string]any{testDraft(30, nil, "")}, nil)
	})

	if err := e.Run(NewCmdDraft, "draft", "list", "--all", "--fields", "page_number"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	want := `{"page_number":12}` + "\n" + `{"page_number":30}` + "\n"
	if got := e.Stdout.String(); got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestList_Empty(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{}, nil)
	})

	if err := e.Run(NewCmdDraft, "draft", "list"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if got, want := e.Stdout.String(), `{"items":[],"next_cursor":null}`+"\n"; got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}

	// 人間向けの表示では標準出力に何も出さず、標準エラーに案内とスコープのヒントを出す。
	e = newEnv(t)
	e.Handle(http.MethodGet, draftPagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{}, nil)
	})
	if err := e.RunTTY(NewCmdDraft, "draft", "list"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if e.Stdout.Len() > 0 {
		t.Errorf("0件の人間向けの表示で標準出力に出力がある: %q", e.Stdout.String())
	}
	if !strings.Contains(e.Stderr.String(), "topic:read") {
		t.Errorf("標準エラーにスコープのヒントが無い: %q", e.Stderr.String())
	}
}

func TestList_UnknownField(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	err := e.Run(NewCmdDraft, "draft", "list", "--fields", "number")
	_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
	assertNoRequest(t, e)
}
