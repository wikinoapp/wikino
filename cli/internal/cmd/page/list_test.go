package page

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// longBodyは、一覧の既定の出力には出さない長い本文。
var longBody = "# 議事録\n\n" + strings.Repeat("あいうえお", 20)

// longExcerptは、APIがlongBodyから作る抜粋。
var longExcerpt = "# 議事録 " + strings.Repeat("あいうえお", 20)[:len("あ")*74] + "…"

type listOutput struct {
	Items      []map[string]any `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

func TestList_JSON(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{
			testListPage(12, ptr("議事録"), longBody, longExcerpt),
			testListPage(5, nil, "短い本文\n", "短い本文"),
		}, ptr("cursor-2"))
	})

	if err := e.Run(NewCmdPage, "page", "list"); err != nil {
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
	if first["excerpt"] != longExcerpt {
		t.Errorf("excerpt = %q、期待値 = %q", first["excerpt"], longExcerpt)
	}
	if first["url"] != "https://wikino.example.com/s/team/pages/12" || first["title"] != "議事録" {
		t.Errorf("1件目 = %v", first)
	}
	if first["topic_name"] != "開発" {
		t.Errorf("topic_name = %v、期待値 = 開発", first["topic_name"])
	}
	for _, key := range []string{"id", "number", "status", "topic_number", "modified_at"} {
		if _, ok := first[key]; !ok {
			t.Errorf("1件目に %s が無い: %v", key, first)
		}
	}
	if second := got.Items[1]; second["title"] != nil || second["excerpt"] != "短い本文" {
		t.Errorf("2件目 = %v", second)
	}
	if got.NextCursor == nil || *got.NextCursor != "cursor-2" {
		t.Errorf("next_cursor = %v、期待値 = cursor-2", got.NextCursor)
	}
	if !strings.Contains(e.Stderr.String(), "--cursor cursor-2") {
		t.Errorf("標準エラーに次のページの取り方が無い: %q", e.Stderr.String())
	}
	// 本文を出さないため、APIに本文を求めない。
	e.AssertQuery(t, "include")
}

func TestList_Params(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{}, nil)
	})

	err := e.Run(NewCmdPage, "page", "list",
		"--topic", "3", "--status", "all", "--title", "議事録",
		"--modified-since", "2026-10-01T09:00:00.5+09:00", "--limit", "50", "--cursor", "abc")
	if err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	requests := e.Requests()
	if len(requests) != 1 {
		t.Fatalf("要求の数 = %d、期待値 = 1", len(requests))
	}
	query := requests[0].URL.Query()
	want := map[string]string{
		"topic_number":   "3",
		"status":         "all",
		"title":          "議事録",
		"modified_since": "2026-10-01T09:00:00.5+09:00",
		"limit":          "50",
		"cursor":         "abc",
	}
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Errorf("クエリの %s = %q、期待値 = %q", key, got, value)
		}
	}
}

func TestList_FieldsBody(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{testListPage(12, ptr("議事録"), "本文\n", "本文")}, nil)
	})

	if err := e.Run(NewCmdPage, "page", "list", "--fields", "number,body"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	want := `{"items":[{"body":"本文\n","number":12}],"next_cursor":null}` + "\n"
	if got := e.Stdout.String(); got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
	e.AssertQuery(t, "include", "body")
}

func TestList_FieldsTopicName(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{testListPage(12, ptr("議事録"), "本文\n", "本文")}, nil)
	})

	if err := e.Run(NewCmdPage, "page", "list", "--fields", "title,topic_name"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	want := `{"items":[{"title":"議事録","topic_name":"開発"}],"next_cursor":null}` + "\n"
	if got := e.Stdout.String(); got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestList_Empty(t *testing.T) {
	t.Parallel()

	t.Run("JSONでは空の一覧を出す", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
			return listResponse([]map[string]any{}, nil)
		})

		if err := e.Run(NewCmdPage, "page", "list"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		if got, want := e.Stdout.String(), `{"items":[],"next_cursor":null}`+"\n"; got != want {
			t.Errorf("標準出力 = %q、期待値 = %q", got, want)
		}
		if e.Stderr.Len() > 0 {
			t.Errorf("標準エラーに出力がある: %q", e.Stderr.String())
		}
	})

	t.Run("人間向けの表示では標準エラーに案内を出す", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
			return listResponse([]map[string]any{}, nil)
		})

		if err := e.RunTTY(NewCmdPage, "page", "list"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		if e.Stdout.Len() > 0 {
			t.Errorf("標準出力に出力がある: %q", e.Stdout.String())
		}
		if got, want := e.Stderr.String(), "該当するものはありません。\n"; got != want {
			t.Errorf("標準エラー = %q、期待値 = %q", got, want)
		}
	})
}

func TestList_All(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("cursor") == "cursor-2" {
			return listResponse([]map[string]any{testListPage(3, ptr("3件目"), "本文", "本文")}, nil)
		}
		return listResponse([]map[string]any{
			testListPage(1, ptr("1件目"), "本文", "本文"),
			testListPage(2, ptr("2件目"), "本文", "本文"),
		}, ptr("cursor-2"))
	})

	if err := e.Run(NewCmdPage, "page", "list", "--all", "--limit", "2", "--fields", "number,title"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	want := []string{
		`{"number":1,"title":"1件目"}`,
		`{"number":2,"title":"2件目"}`,
		`{"number":3,"title":"3件目"}`,
	}
	got := cmdtest.Lines(e.Stdout.String())
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("標準出力の行 = %q、期待値 = %q", got, want)
	}

	requests := e.Requests()
	if len(requests) != 2 {
		t.Fatalf("要求の数 = %d、期待値 = 2", len(requests))
	}
	if q := requests[1].URL.Query(); q.Get("cursor") != "cursor-2" || q.Get("limit") != "2" {
		t.Errorf("2回目の要求のクエリ = %v", q)
	}
	if e.Stderr.Len() > 0 {
		t.Errorf("全件の取得で標準エラーに出力がある: %q", e.Stderr.String())
	}
}

func TestList_AllOnTTY(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{testListPage(1, ptr("1件目"), "本文", "本文")}, nil)
	})

	if err := e.RunTTY(NewCmdPage, "page", "list", "--all"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	lines := cmdtest.Lines(e.Stdout.String())
	if len(lines) != 1 || !strings.HasPrefix(lines[0], `{"excerpt":"本文"`) {
		t.Errorf("TTYでも1行1件のJSONを期待したが %q だった", e.Stdout.String())
	}
}

func TestList_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		unpublished := testListPage(5, nil, "", "")
		unpublished["status"] = "unpublished"
		return listResponse([]map[string]any{testListPage(12, ptr("議事録"), longBody, longExcerpt), unpublished}, ptr("cursor-2"))
	})

	if err := e.RunTTY(NewCmdPage, "page", "list"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "list", e.Stdout.String())
	if !strings.Contains(e.Stderr.String(), "--cursor cursor-2") {
		t.Errorf("標準エラーに次のページの取り方が無い: %q", e.Stderr.String())
	}
}

func TestList_HumanTopicNameControlCharacters(t *testing.T) {
	t.Parallel()

	// トピック名は他の利用者が付けた値で、端末を操作するエスケープシーケンスや改行を含みうる。
	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		page := testListPage(12, ptr("議事録"), "", "")
		page["topic_name"] = "開\x1b[2J発\n偽の行"
		return listResponse([]map[string]any{page}, nil)
	})

	if err := e.RunTTY(NewCmdPage, "page", "list"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	got := e.Stdout.String()
	if strings.Contains(got, "\x1b") || !strings.Contains(got, "開 [2J発 偽の行  3 ") {
		t.Errorf("トピック名の制御文字が1行の値として無害化されていない: %q", got)
	}
}

func TestList_InvalidFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"トピック番号が0", []string{"--topic", "0"}},
		{"トピック番号にクエリが付いている", []string{"--topic", "3?x=1"}},
		{"未知の公開状態", []string{"--status", "draft"}},
		{"RFC 3339でない日時", []string{"--modified-since", "2026-10-01"}},
		{"件数が0", []string{"--limit", "0"}},
		{"全件とカーソルの併用", []string{"--all", "--cursor", "abc"}},
		{"未知の項目", []string{"--fields", "number,content"}},
		{"位置引数", []string{"12"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdPage, append([]string{"page", "list"}, tt.args...)...)
			_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			assertNoRequest(t, e)
		})
	}
}

func TestList_Forbidden(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusForbidden, "スコープが足りません。")
	})

	err := e.Run(NewCmdPage, "page", "list")
	_ = cmdtest.AssertCLIError(t, err, output.CodeForbidden, output.ExitForbidden)
	if e.Stdout.Len() > 0 {
		t.Errorf("失敗したときに標準出力に出力がある: %q", e.Stdout.String())
	}
}
