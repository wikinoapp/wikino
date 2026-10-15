package page

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// linkCommandsは、ページを起点にした一覧のコマンドと、それぞれが呼ぶAPIのパス。
// 2つは呼ぶAPIだけが違うため、同じテストを両方に当てる。
var linkCommands = []struct {
	name      string
	path      string
	emptyHint string
}{
	{"links", pagesPath + "/12/links", "公開済みの本文のWikiリンクだけが対象です。下書きにだけあるリンクは含みません。"},
	{"backlinks", pagesPath + "/12/backlinks", "公開済みの本文のWikiリンクだけが対象です。下書きにだけあるリンクは含みません。"},
}

func TestLinks_JSON(t *testing.T) {
	t.Parallel()

	for _, c := range linkCommands {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodGet, c.path, func(req *http.Request) *http.Response {
				unpublished := testPage(5, ptr("定例会議"), "")
				unpublished["status"] = "unpublished"
				return listResponse([]map[string]any{testPage(7, ptr("予算案"), "予算の\n確認"), unpublished}, ptr("cursor-2"))
			})

			if err := e.Run(NewCmdPage, "page", c.name, "12", "--limit", "2"); err != nil {
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
			if first["excerpt"] != "予算の 確認" || first["url"] != "https://wikino.example.com/s/team/pages/7" {
				t.Errorf("1件目 = %v", first)
			}
			if got.Items[1]["status"] != "unpublished" {
				t.Errorf("2件目 = %v", got.Items[1])
			}
			if got.NextCursor == nil || *got.NextCursor != "cursor-2" {
				t.Errorf("next_cursor = %v、期待値 = cursor-2", got.NextCursor)
			}
			if !strings.Contains(e.Stderr.String(), "--cursor cursor-2") {
				t.Errorf("標準エラーに次のページの取り方が無い: %q", e.Stderr.String())
			}
			if q := e.Requests()[0].URL.Query(); q.Get("limit") != "2" || q.Has("cursor") {
				t.Errorf("要求のクエリ = %v", q)
			}
		})
	}
}

func TestLinks_Human(t *testing.T) {
	t.Parallel()

	for _, c := range linkCommands {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodGet, c.path, func(req *http.Request) *http.Response {
				unpublished := testPage(5, nil, "")
				unpublished["status"] = "unpublished"
				return listResponse([]map[string]any{testPage(7, ptr("予算案"), longBody), unpublished}, nil)
			})

			if err := e.RunTTY(NewCmdPage, "page", c.name, "12"); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			cmdtest.AssertGolden(t, c.name, e.Stdout.String())
			if e.Stderr.Len() > 0 {
				t.Errorf("標準エラーに出力がある: %q", e.Stderr.String())
			}
		})
	}
}

func TestLinks_All(t *testing.T) {
	t.Parallel()

	for _, c := range linkCommands {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodGet, c.path, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("cursor") == "cursor-2" {
					return listResponse([]map[string]any{testPage(3, ptr("3件目"), "本文")}, nil)
				}
				return listResponse([]map[string]any{testPage(1, ptr("1件目"), "本文"), testPage(2, ptr("2件目"), "本文")}, ptr("cursor-2"))
			})

			if err := e.Run(NewCmdPage, "page", c.name, "12", "--all", "--fields", "number,title"); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			want := []string{
				`{"number":1,"title":"1件目"}`,
				`{"number":2,"title":"2件目"}`,
				`{"number":3,"title":"3件目"}`,
			}
			if got := cmdtest.Lines(e.Stdout.String()); strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("標準出力の行 = %q、期待値 = %q", got, want)
			}
			if len(e.Requests()) != 2 {
				t.Errorf("要求の数 = %d、期待値 = 2", len(e.Requests()))
			}
		})
	}
}

func TestLinks_Empty(t *testing.T) {
	t.Parallel()

	for _, c := range linkCommands {
		t.Run(c.name+": JSONでは空の一覧を出す", func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodGet, c.path, func(req *http.Request) *http.Response {
				return listResponse([]map[string]any{}, nil)
			})

			if err := e.Run(NewCmdPage, "page", c.name, "12"); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if got, want := e.Stdout.String(), `{"items":[],"next_cursor":null}`+"\n"; got != want {
				t.Errorf("標準出力 = %q、期待値 = %q", got, want)
			}
			if e.Stderr.Len() > 0 {
				t.Errorf("標準エラーに出力がある: %q", e.Stderr.String())
			}
		})

		t.Run(c.name+": 人間向けの表示では標準エラーに案内とヒントを出す", func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodGet, c.path, func(req *http.Request) *http.Response {
				return listResponse([]map[string]any{}, nil)
			})

			if err := e.RunTTY(NewCmdPage, "page", c.name, "12"); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if e.Stdout.Len() > 0 {
				t.Errorf("標準出力に出力がある: %q", e.Stdout.String())
			}
			if got, want := e.Stderr.String(), "該当するものはありません。\n"+c.emptyHint+"\n"; got != want {
				t.Errorf("標準エラー = %q、期待値 = %q", got, want)
			}
		})
	}
}

func TestLinks_NotFound(t *testing.T) {
	t.Parallel()

	for _, c := range linkCommands {
		for _, all := range []bool{false, true} {
			name := c.name
			args := []string{"page", c.name, "12"}
			if all {
				name += " --all"
				args = append(args, "--all")
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				e := newEnv(t)
				e.Handle(http.MethodGet, c.path, func(req *http.Request) *http.Response {
					return cmdtest.Problem(http.StatusNotFound, "ページが見つかりません。")
				})

				err := e.Run(NewCmdPage, args...)
				cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
				if !strings.Contains(cliErr.Hint, "wikino page list") {
					t.Errorf("ヒントにページの一覧のコマンドが無い: %q", cliErr.Hint)
				}
				if e.Stdout.Len() > 0 {
					t.Errorf("失敗したときに標準出力に出力がある: %q", e.Stdout.String())
				}
			})
		}
	}
}

func TestLinks_InvalidArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"ページ番号が無い", nil},
		{"ページ番号が0", []string{"0"}},
		{"ページ番号が整数でない", []string{"abc"}},
		{"ページ番号が2つ", []string{"12", "13"}},
		{"件数が0", []string{"12", "--limit", "0"}},
		{"全件とカーソルの併用", []string{"12", "--all", "--cursor", "abc"}},
		{"未知の項目", []string{"12", "--fields", "number,content"}},
	}
	for _, c := range linkCommands {
		for _, tt := range tests {
			t.Run(c.name+": "+tt.name, func(t *testing.T) {
				t.Parallel()

				e := newEnv(t)
				err := e.Run(NewCmdPage, append([]string{"page", c.name}, tt.args...)...)
				_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
				assertNoRequest(t, e)
			})
		}
	}
}
