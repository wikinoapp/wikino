package page

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const etag = `"d41d8cd9"`

func handlePage(e *cmdtest.Env, number int32, page map[string]any) {
	e.Handle(http.MethodGet, pagesPath+"/"+strconv.Itoa(int(number)), func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, page, "ETag", etag)
	})
}

func TestGet_JSON(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePage(e, 12, testPage(12, ptr("議事録"), "# 議事録\n\n本文\n"))

	if err := e.Run(NewCmdPage, "page", "get", "12"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	var got map[string]any
	cmdtest.Decode(t, e.Stdout.String(), &got)
	want := map[string]any{
		"etag":  etag,
		"url":   "https://wikino.example.com/s/team/pages/12",
		"body":  "# 議事録\n\n本文\n",
		"title": "議事録",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %v、期待値 = %v", key, got[key], value)
		}
	}
	if _, ok := got["excerpt"]; ok {
		t.Errorf("1件の出力に抜粋が含まれる: %v", got)
	}
}

func TestGet_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePage(e, 12, testPage(12, ptr("議事録"), "# 議事録\n\n- [ ] 予算案\n"))

	if err := e.RunTTY(NewCmdPage, "page", "get", "12"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "get", e.Stdout.String())
}

// pageFileCaseは、リポジトリ直下の共通のテストデータ (`testdata/wikino-file-format/page-files.json`) の1件。
// `--md` の出力が、エクスポートと同じページファイルになることを確かめる。
type pageFileCase struct {
	Name string         `json:"name"`
	Page map[string]any `json:"page"`
	Want string         `json:"want"`
}

func TestGet_MarkdownSharedTestdata(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "wikino-file-format", "page-files.json"))
	if err != nil {
		t.Fatalf("共通のテストデータを読めません: %v", err)
	}
	var cases []pageFileCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("共通のテストデータを解釈できません: %v", err)
	}

	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()

			// テストデータのページは、公開APIのページの応答の項目に `etag` を加えたもの。
			page := map[string]any{"id": "00000000-0000-0000-0000-000000000001"}
			for key, value := range tt.Page {
				page[key] = value
			}
			number := strconv.Itoa(int(page["number"].(float64)))

			e := newEnv(t)
			e.Handle(http.MethodGet, pagesPath+"/"+number, func(req *http.Request) *http.Response {
				return cmdtest.JSON(http.StatusOK, page, "ETag", page["etag"].(string))
			})

			// TTYでも非TTYでも同じページファイルを出す。
			for _, run := range []func() error{
				func() error { return e.Run(NewCmdPage, "page", "get", number, "--md") },
				func() error { return e.RunTTY(NewCmdPage, "page", "get", number, "--md") },
			} {
				if err := run(); err != nil {
					t.Fatalf("エラーを期待しなかったが %v だった", err)
				}
				if got := e.Stdout.String(); got != tt.Want {
					t.Errorf("標準出力 = %q、期待値 = %q", got, tt.Want)
				}
			}
		})
	}
}

func TestGet_MarkdownWithOutputFlags(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"--json"},
		{"--fields", "number"},
		{"--jq", ".body"},
	} {
		t.Run(args[0], func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdPage, append([]string{"page", "get", "12", "--md"}, args...)...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, "--md, "+args[0]) {
				t.Errorf("メッセージに併用したフラグが無い: %q", cliErr.Message)
			}
			assertNoRequest(t, e)
		})
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath+"/99", func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "ページが見つかりません。")
	})

	err := e.Run(NewCmdPage, "page", "get", "99")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "wikino page list") {
		t.Errorf("ヒントにページの一覧のコマンドが無い: %q", cliErr.Hint)
	}
	if cliErr.Status != http.StatusNotFound {
		t.Errorf("status = %d、期待値 = 404", cliErr.Status)
	}
}

func TestGet_InvalidNumber(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"0", "abc", "12?x=1", "../12", "%31%32", "12\n"} {
		t.Run(arg, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdPage, "page", "get", arg)
			_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			assertNoRequest(t, e)
		})
	}
}
