package page

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const (
	pagePath = pagesPath + "/12"
	// newETagは、更新後のページの版。
	newETag = `"9e107d9d"`
)

// handleUpdateは、ページ12の更新の要求を受けて記録し、更新したページを返す。
// 本文を書き込むときのページの取得には版etagのページを、Wikiリンクのプレビューにはリンクの無い結果を返す。
func handleUpdate(t *testing.T, e *cmdtest.Env, sent *sentRequest) {
	t.Helper()

	handleCurrentPage(e, etag)
	handlePreview(t, e)
	e.Handle(http.MethodPatch, pagePath, func(req *http.Request) *http.Response {
		recordRequest(t, req, sent)
		page := testPage(12, ptr("議事録"), "# 議事録\n")
		for key, value := range sent.body {
			page[key] = value
		}
		return cmdtest.JSON(http.StatusOK, page, "ETag", newETag)
	})
}

func TestUpdate_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantBody map[string]any
	}{
		{
			name:     "本文だけ",
			args:     []string{"--body", "# 議事録\n\n追記\n"},
			wantBody: map[string]any{"body": "# 議事録\n\n追記\n"},
		},
		{
			name:     "タイトルだけ",
			args:     []string{"--title", "定例会議の議事録"},
			wantBody: map[string]any{"title": "定例会議の議事録"},
		},
		{
			name:     "タイトルと本文",
			args:     []string{"--title", "定例会議の議事録", "--body", ""},
			wantBody: map[string]any{"title": "定例会議の議事録", "body": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			var sent sentRequest
			handleUpdate(t, e, &sent)

			args := append([]string{"page", "update", "12", "--if-match", etag}, tt.args...)
			if err := e.Run(NewCmdPage, args...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			if sent.ifMatch != etag {
				t.Errorf("If-Match = %q、期待値 = %q", sent.ifMatch, etag)
			}
			if sent.contentType != "application/merge-patch+json" {
				t.Errorf("Content-Type = %q、期待値 = application/merge-patch+json", sent.contentType)
			}
			// 指定しなかった項目は送らず、変えない。
			if len(sent.body) != len(tt.wantBody) {
				t.Errorf("送った本文 = %v、期待値 = %v", sent.body, tt.wantBody)
			}
			for key, value := range tt.wantBody {
				if sent.body[key] != value {
					t.Errorf("送った %s = %v、期待値 = %v", key, sent.body[key], value)
				}
			}

			var got map[string]any
			cmdtest.Decode(t, e.Stdout.String(), &got)
			if got["etag"] != newETag {
				t.Errorf("出力のetag = %v、期待値 = %s (更新後の版)", got["etag"], newETag)
			}
		})
	}
}

func TestUpdate_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	var sent sentRequest
	handleUpdate(t, e, &sent)

	if err := e.RunTTY(NewCmdPage, "page", "update", "12", "--if-match", etag, "--body", "# 議事録\n\n追記\n"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "update", e.Stdout.String())
	if got, want := e.Stderr.String(), "ページ12を更新しました。\n"; got != want {
		t.Errorf("標準エラー = %q、期待値 = %q", got, want)
	}
}

func TestUpdate_BodyFromStdin(t *testing.T) {
	t.Parallel()

	const body = "# 議事録\n\n追記\n\n\n"
	e := newEnv(t)
	e.Stdin = body
	var sent sentRequest
	handleUpdate(t, e, &sent)

	if err := e.Run(NewCmdPage, "page", "update", "12", "--if-match", etag, "--body-file", "-"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if got := sent.body["body"]; got != body {
		t.Errorf("送った本文 = %q、期待値 = %q (末尾の改行を保つ)", got, body)
	}
}

func TestUpdate_Force(t *testing.T) {
	t.Parallel()

	const latestETag = `"latest"`
	e := newEnv(t)
	var sent sentRequest
	handleUpdate(t, e, &sent)
	handleCurrentPage(e, latestETag)

	if err := e.Run(NewCmdPage, "page", "update", "12", "--force", "--body", "上書き\n"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// 最新の版を取得し、Wikiリンクを確かめてから、その版で更新する。
	requests := e.Requests()
	if len(requests) != 3 || requests[0].Method != http.MethodGet || requests[1].URL.Path != previewPath || requests[2].Method != http.MethodPatch {
		t.Fatalf("要求の順 = %v、期待値 = GET → プレビュー → PATCH", methods(requests))
	}
	if sent.ifMatch != latestETag {
		t.Errorf("If-Match = %q、期待値 = %q (取得した最新の版)", sent.ifMatch, latestETag)
	}
}

func TestUpdate_ForceNotFound(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagePath, func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "ページが見つかりません。")
	})

	err := e.Run(NewCmdPage, "page", "update", "12", "--force", "--body", "上書き\n")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "wikino page list") {
		t.Errorf("ヒントにページの一覧のコマンドが無い: %q", cliErr.Hint)
	}
	if requests := e.Requests(); len(requests) != 1 {
		t.Errorf("要求 = %v、期待値 = 取得の1件だけ", methods(requests))
	}
}

func TestUpdate_UsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantMessage string
		wantHint    string
	}{
		{
			name:        "版の指定が無い",
			args:        []string{"12", "--body", "本文"},
			wantMessage: "--if-match",
			wantHint:    "`wikino page get 12`",
		},
		{
			name:        "--if-matchと--forceの併用",
			args:        []string{"12", "--body", "本文", "--if-match", etag, "--force"},
			wantMessage: "--if-match, --force",
		},
		{
			name:        "空の--if-match",
			args:        []string{"12", "--body", "本文", "--if-match", ""},
			wantMessage: "--if-match",
			wantHint:    "`wikino page get 12`",
		},
		{
			name:        "改行を含む--if-match",
			args:        []string{"12", "--body", "本文", "--if-match", etag + "\nX-Injected: 1"},
			wantMessage: "--if-match",
		},
		{
			name:        "版を問わない--if-match",
			args:        []string{"12", "--body", "本文", "--if-match", "*"},
			wantMessage: "`*`",
			wantHint:    "--force",
		},
		{
			name:        "更新する項目が無い",
			args:        []string{"12", "--if-match", etag},
			wantMessage: "--title",
		},
		{
			name:        "本文のフラグの併用",
			args:        []string{"12", "--if-match", etag, "--body", "a", "--body-file", "-"},
			wantMessage: "--body, --body-file",
		},
		{
			name:        "不正なページ番号",
			args:        []string{"../12", "--if-match", etag, "--body", "本文"},
			wantMessage: "<page_number>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdPage, append([]string{"page", "update"}, tt.args...)...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, tt.wantMessage) {
				t.Errorf("メッセージ %q に %q が無い", cliErr.Message, tt.wantMessage)
			}
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒント %q に %q が無い", cliErr.Hint, tt.wantHint)
			}
			assertNoRequest(t, e)
		})
	}
}

func TestUpdate_DryRun(t *testing.T) {
	t.Parallel()

	t.Run("--if-match", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		if err := e.Run(NewCmdPage, "page", "update", "12", "--if-match", etag, "--title", "定例会議", "--dry-run"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		assertNoRequest(t, e)

		got := decodeDryRun(t, e.Stdout.String())
		if len(got.Requests) != 1 {
			t.Fatalf("記録した要求 = %d 件、期待値 = 1 件", len(got.Requests))
		}
		req := got.Requests[0]
		if req.Method != http.MethodPatch || req.URL != cmdtest.Host+pagePath {
			t.Errorf("要求 = %s %s", req.Method, req.URL)
		}
		if req.Headers["If-Match"] != etag {
			t.Errorf("要求のIf-Match = %q、期待値 = %q", req.Headers["If-Match"], etag)
		}
		if len(req.Body) != 1 || req.Body["title"] != "定例会議" {
			t.Errorf("要求の本文 = %v", req.Body)
		}
		assertNoSecret(t, e)
	})

	t.Run("--force", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		handleCurrentPage(e, etag)
		handlePreview(t, e)
		if err := e.Run(NewCmdPage, "page", "update", "12", "--force", "--body", "上書き\n", "--dry-run"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}

		// 版の取得とWikiリンクのプレビュー (読み取り) だけを送り、更新の要求は送らない。
		if requests := e.Requests(); len(requests) != 2 || requests[0].Method != http.MethodGet || requests[1].URL.Path != previewPath {
			t.Errorf("送った要求 = %v、期待値 = GET → プレビュー", methods(requests))
		}
		got := decodeDryRun(t, e.Stdout.String())
		if len(got.Requests) != 1 || got.Requests[0].Method != http.MethodPatch || got.Requests[0].Headers["If-Match"] != etag {
			t.Errorf("記録した要求 = %+v", got.Requests)
		}
	})
}

func TestUpdate_APIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		status       int
		wantCode     string
		wantExitCode int
		wantHint     string
	}{
		{name: "412", status: http.StatusPreconditionFailed, wantCode: output.CodePreconditionFailed, wantExitCode: output.ExitConflict, wantHint: "`wikino page get 12`"},
		{name: "428", status: http.StatusPreconditionRequired, wantCode: output.CodePreconditionRequired, wantExitCode: output.ExitConflict, wantHint: "`wikino page get 12`"},
		{name: "422", status: http.StatusUnprocessableEntity, wantCode: output.CodeValidationFailed, wantExitCode: output.ExitValidation, wantHint: "--title"},
		{name: "404", status: http.StatusNotFound, wantCode: output.CodeNotFound, wantExitCode: output.ExitNotFound, wantHint: "wikino page list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			handleCurrentPage(e, etag)
			handlePreview(t, e)
			e.Handle(http.MethodPatch, pagePath, func(req *http.Request) *http.Response {
				return cmdtest.Problem(tt.status, "失敗しました。")
			})

			err := e.Run(NewCmdPage, "page", "update", "12", "--if-match", etag, "--body", "本文")
			cliErr := cmdtest.AssertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒント %q に %q が無い", cliErr.Hint, tt.wantHint)
			}
			if cliErr.Status != tt.status {
				t.Errorf("status = %d、期待値 = %d", cliErr.Status, tt.status)
			}
		})
	}
}

func methods(requests []*http.Request) []string {
	got := make([]string, 0, len(requests))
	for _, req := range requests {
		got = append(got, req.Method+" "+req.URL.Path)
	}

	return got
}

func TestUpdate_DryRunWithFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "読んだ版の指定", args: []string{"--if-match", etag}},
		{name: "最新の版の取得も拒否", args: []string{"--force"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			args := append([]string{"page", "update", "12", "--body", "本文", "--dry-run", "--fields", "number,etag"}, tt.args...)
			err := e.Run(NewCmdPage, args...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, "--dry-run, --fields") || !strings.Contains(cliErr.Hint, "`--fields` を外して") {
				t.Errorf("併用できないフラグと外すフラグの案内が無い: %v", cliErr)
			}
			assertNoRequest(t, e)
			if e.Stdout.Len() > 0 {
				t.Errorf("拒否時の標準出力に結果がある: %q", e.Stdout.String())
			}
		})
	}
}
