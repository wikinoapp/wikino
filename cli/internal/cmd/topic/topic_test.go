package topic

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const topicsPath = "/api/v1/spaces/team/topics"

func testTopic(number int32, name, visibility string) map[string]any {
	return map[string]any{
		"id":          fmt.Sprintf("00000000-0000-0000-0000-%012d", number),
		"number":      number,
		"name":        name,
		"description": name + "の説明",
		"visibility":  visibility,
	}
}

func handleList(e *cmdtest.Env) {
	e.Handle(http.MethodGet, topicsPath, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("cursor") == "cursor-2" {
			return cmdtest.JSON(http.StatusOK, map[string]any{
				"items":       []map[string]any{testTopic(3, "設計", "private")},
				"next_cursor": nil,
			})
		}
		return cmdtest.JSON(http.StatusOK, map[string]any{
			"items":       []map[string]any{testTopic(1, "全体", "public"), testTopic(2, "議事録", "public")},
			"next_cursor": "cursor-2",
		})
	})
}

func TestList_JSON(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleList(e)

	if err := e.Run(NewCmdTopic, "topic", "list", "--limit", "2"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	var got struct {
		Items      []map[string]any `json:"items"`
		NextCursor *string          `json:"next_cursor"`
	}
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if len(got.Items) != 2 || got.Items[0]["name"] != "全体" || got.Items[0]["description"] != "全体の説明" {
		t.Errorf("items = %v", got.Items)
	}
	if got.NextCursor == nil || *got.NextCursor != "cursor-2" {
		t.Errorf("next_cursor = %v、期待値 = cursor-2", got.NextCursor)
	}
	if q := e.Requests()[0].URL.Query(); q.Get("limit") != "2" {
		t.Errorf("クエリ = %v", q)
	}
	if !strings.Contains(e.Stderr.String(), "--cursor cursor-2") {
		t.Errorf("標準エラーに次のページの取り方が無い: %q", e.Stderr.String())
	}
}

func TestList_All(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleList(e)

	if err := e.Run(NewCmdTopic, "topic", "list", "--all", "--fields", "number,name"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	want := `{"name":"全体","number":1}` + "\n" + `{"name":"議事録","number":2}` + "\n" + `{"name":"設計","number":3}` + "\n"
	if got := e.Stdout.String(); got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestList_Empty(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	e.Handle(http.MethodGet, topicsPath, func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, map[string]any{"items": []any{}, "next_cursor": nil})
	})

	if err := e.Run(NewCmdTopic, "topic", "list"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if got, want := e.Stdout.String(), `{"items":[],"next_cursor":null}`+"\n"; got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestList_Human(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleList(e)

	if err := e.RunTTY(NewCmdTopic, "topic", "list"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "list", e.Stdout.String())
}

func TestGet_JSON(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	e.Handle(http.MethodGet, topicsPath+"/3", func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testTopic(3, "設計", "private"))
	})

	if err := e.Run(NewCmdTopic, "topic", "get", "3"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	want := `{"description":"設計の説明","id":"00000000-0000-0000-0000-000000000003","name":"設計","number":3,"visibility":"private"}` + "\n"
	if got := e.Stdout.String(); got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestGet_Human(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	e.Handle(http.MethodGet, topicsPath+"/3", func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testTopic(3, "設計", "private"))
	})

	if err := e.RunTTY(NewCmdTopic, "topic", "get", "3"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "get", e.Stdout.String())
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	e.Handle(http.MethodGet, topicsPath+"/9", func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "トピックが見つかりません。")
	})

	err := e.Run(NewCmdTopic, "topic", "get", "9")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "wikino topic list") {
		t.Errorf("ヒントにトピックの一覧のコマンドが無い: %q", cliErr.Hint)
	}
}

func TestGet_InvalidNumber(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"0", "abc", "3#x", "1.5"} {
		t.Run(arg, func(t *testing.T) {
			t.Parallel()

			e := cmdtest.New(t)
			err := e.Run(NewCmdTopic, "topic", "get", arg)
			_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if len(e.Requests()) > 0 {
				t.Errorf("要求を送らないことを期待したが送った")
			}
		})
	}
}

func TestGroup_RequiresSubcommand(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"topic"}, {"topic", "show"}} {
		e := cmdtest.New(t)
		err := e.Run(NewCmdTopic, args...)
		_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
	}
}
