package draft

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const revisionsPath = pagesPath + "/12/draft_page_revisions"

// handleRevisionsは、ページ12の下書きの編集履歴の一覧の要求に、counts[i]件ずつのページで応答する。
func handleRevisions(t *testing.T, e *cmdtest.Env, counts ...int) {
	t.Helper()

	e.Handle(http.MethodGet, revisionsPath, func(req *http.Request) *http.Response {
		page := 0
		if cursor := req.URL.Query().Get("cursor"); cursor != "" {
			if _, err := fmt.Sscanf(cursor, "page-%d", &page); err != nil {
				t.Errorf("想定外のカーソル: %q", cursor)
			}
		}
		if got := req.URL.Query().Get("limit"); got != "100" {
			t.Errorf("limit = %q、期待値 = 100", got)
		}

		items := make([]map[string]any, 0, counts[page])
		for i := range counts[page] {
			items = append(items, map[string]any{
				"id":         fmt.Sprintf("00000000-0000-0000-0000-%012d", page*100+i),
				"title":      "議事録",
				"body":       "# 議事録\n",
				"created_at": "2026-10-01T09:00:00Z",
			})
		}
		var next *string
		if page+1 < len(counts) {
			next = ptr(fmt.Sprintf("page-%d", page+1))
		}
		return cmdtest.JSON(http.StatusOK, map[string]any{"items": items, "next_cursor": next})
	})
}

// handleDeleteは、ページ12の下書きの削除の要求を受けて送った版をifMatchに記録し、204で応答する。
func handleDelete(e *cmdtest.Env, ifMatch *string) {
	e.Handle(http.MethodDelete, draftPath(12), func(req *http.Request) *http.Response {
		*ifMatch = req.Header.Get("If-Match")
		return &http.Response{StatusCode: http.StatusNoContent, Header: http.Header{}, Body: http.NoBody}
	})
}

// confirmationOutputは、確認プロトコルのJSON。
type confirmationOutput struct {
	Status         string   `json:"status"`
	Changes        []string `json:"changes"`
	ConfirmCommand string   `json:"confirm_command"`
}

// assertConfirmationRequiredは、削除の要求を送らずに終了コード9で確認プロトコルのJSONを出したことを確かめる。
func assertConfirmationRequired(t *testing.T, e *cmdtest.Env, err error) confirmationOutput {
	t.Helper()

	var silent *output.SilentError
	if !errors.As(err, &silent) || silent.ExitCode != output.ExitConfirmationRequired {
		t.Fatalf("終了コード9を期待したが %v だった", err)
	}
	for _, path := range requestPaths(e) {
		if strings.HasPrefix(path, http.MethodDelete) {
			t.Errorf("確認の前に削除の要求を送った: %s", path)
		}
	}

	var got confirmationOutput
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if got.Status != "confirmation_required" {
		t.Errorf("status = %q、期待値 = confirmation_required", got.Status)
	}
	assertNoSecret(t, e)

	return got
}

func TestDelete_ConfirmationRequired(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handleCurrentDraft(e, etag)
	// 編集履歴はページ送りを辿って全件を数える。
	handleRevisions(t, e, 100, 3)

	err := e.Run(NewCmdDraft, "draft", "delete", "12", "--if-match", etag, "--space", "team")
	got := assertConfirmationRequired(t, e, err)

	wantChanges := []string{
		"ページ12「議事録」の自分の下書き (更新日時: 2026-10-01T09:00:00Z) を削除します",
		"下書きの編集履歴103件も削除します。削除した下書きと編集履歴は元に戻せません",
		"ページそのものと公開済みの内容は変わりません",
	}
	if strings.Join(got.Changes, "\n") != strings.Join(wantChanges, "\n") {
		t.Errorf("changes = %q、期待値 = %q", got.Changes, wantChanges)
	}
	// 承認後のコマンドは、確認したときと同じスペースで、取得した版を固定して実行する。
	if want := `wikino draft delete 12 --if-match '"d41d8cd9"' --space team --yes`; got.ConfirmCommand != want {
		t.Errorf("confirm_command = %q、期待値 = %q", got.ConfirmCommand, want)
	}
}

func TestDelete_ConfirmationUntitled(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPath(12), func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testDraft(12, nil, ""), "ETag", etag)
	})
	handleRevisions(t, e, 0)

	err := e.Run(NewCmdDraft, "draft", "delete", "12", "--if-match", etag)
	got := assertConfirmationRequired(t, e, err)
	if len(got.Changes) < 2 || !strings.Contains(got.Changes[0], "「(タイトルなし)」") || !strings.Contains(got.Changes[1], "編集履歴0件") {
		t.Errorf("changes = %q", got.Changes)
	}
}

func TestDelete_ConfirmationWithForce(t *testing.T) {
	t.Parallel()

	const latestETag = `"latest"`
	e := newEnv(t)
	handleCurrentDraft(e, latestETag)
	handleRevisions(t, e, 1)

	err := e.Run(NewCmdDraft, "draft", "delete", "12", "--force")
	got := assertConfirmationRequired(t, e, err)

	// `--force` で取得した版を固定し、`--force` は引き継がない。
	if want := `wikino draft delete 12 --if-match '"latest"' --yes`; got.ConfirmCommand != want {
		t.Errorf("confirm_command = %q、期待値 = %q", got.ConfirmCommand, want)
	}
}

func TestDelete_VersionChangedBeforeConfirmation(t *testing.T) {
	t.Parallel()

	// 読んだ版の後に下書きが更新されていれば、読んでいない内容の削除を確かめさせずに失敗させる。
	e := newEnv(t)
	handleCurrentDraft(e, `"latest"`)

	err := e.Run(NewCmdDraft, "draft", "delete", "12", "--if-match", etag)
	cliErr := cmdtest.AssertCLIError(t, err, output.CodePreconditionFailed, output.ExitConflict)
	if !strings.Contains(cliErr.Hint, "`wikino draft get 12`") {
		t.Errorf("ヒントに今の内容を確かめるコマンドが無い: %q", cliErr.Hint)
	}
	if got := requestPaths(e); len(got) != 1 {
		t.Errorf("要求 = %v、期待値 = 下書きの取得の1件だけ", got)
	}
	if e.Stdout.Len() > 0 {
		t.Errorf("標準出力に確認が出た: %q", e.Stdout.String())
	}
}

func TestDelete_Yes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		wantPaths []string
		wantETag  string
	}{
		{
			name:      "--if-match",
			args:      []string{"--if-match", etag},
			wantPaths: []string{"DELETE " + draftPath(12)},
			wantETag:  etag,
		},
		{
			name:      "--force",
			args:      []string{"--force"},
			wantPaths: []string{"GET " + draftPath(12), "DELETE " + draftPath(12)},
			wantETag:  `"latest"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			handleCurrentDraft(e, `"latest"`)
			var ifMatch string
			handleDelete(e, &ifMatch)

			args := append([]string{"draft", "delete", "12", "--yes"}, tt.args...)
			if err := e.Run(NewCmdDraft, args...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			// `--yes` では確認のための取得をしない。
			if got := requestPaths(e); strings.Join(got, "|") != strings.Join(tt.wantPaths, "|") {
				t.Errorf("要求 = %v、期待値 = %v", got, tt.wantPaths)
			}
			if ifMatch != tt.wantETag {
				t.Errorf("If-Match = %q、期待値 = %q", ifMatch, tt.wantETag)
			}
			var got map[string]any
			cmdtest.Decode(t, e.Stdout.String(), &got)
			if len(got) != 2 || got["page_number"] != float64(12) || got["deleted"] != true {
				t.Errorf("出力 = %v、期待値 = {page_number: 12, deleted: true}", got)
			}
		})
	}
}

func TestDelete_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	var ifMatch string
	handleDelete(e, &ifMatch)

	if err := e.RunTTY(NewCmdDraft, "draft", "delete", "12", "--if-match", etag, "--yes"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if e.Stdout.Len() > 0 {
		t.Errorf("標準出力 = %q、期待値 = 空", e.Stdout.String())
	}
	if got, want := e.Stderr.String(), "ページ12の下書きを削除しました。\n"; got != want {
		t.Errorf("標準エラー = %q、期待値 = %q", got, want)
	}
}

func TestDelete_Prompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		answer     string
		wantDelete bool
	}{
		{name: "承認", answer: "y\n", wantDelete: true},
		{name: "拒否", answer: "n\n"},
		{name: "入力なし", answer: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Stdin = tt.answer
			handleCurrentDraft(e, etag)
			handleRevisions(t, e, 2)
			var ifMatch string
			handleDelete(e, &ifMatch)

			err := e.RunTTY(NewCmdDraft, "draft", "delete", "12", "--if-match", etag)

			// 端末では、標準エラーに削除する内容を示して尋ねる。
			if !strings.Contains(e.Stderr.String(), "- ページ12「議事録」の自分の下書き") || !strings.Contains(e.Stderr.String(), "編集履歴2件") {
				t.Errorf("標準エラーに確認の内容が無い: %q", e.Stderr.String())
			}
			if tt.wantDelete {
				if err != nil {
					t.Fatalf("エラーを期待しなかったが %v だった", err)
				}
				if ifMatch != etag {
					t.Errorf("If-Match = %q、期待値 = %q", ifMatch, etag)
				}
				return
			}

			_ = cmdtest.AssertCLIError(t, err, output.CodeConfirmationDeclined, output.ExitConfirmationRequired)
			for _, path := range requestPaths(e) {
				if strings.HasPrefix(path, http.MethodDelete) {
					t.Errorf("拒否したのに削除の要求を送った: %s", path)
				}
			}
		})
	}
}

func TestDelete_DryRun(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	if err := e.Run(NewCmdDraft, "draft", "delete", "12", "--if-match", etag, "--dry-run"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// `--dry-run` は何も削除しないため、確認を求めず、取得もしない。
	assertNoRequest(t, e)
	var got struct {
		DryRun   bool `json:"dry_run"`
		Requests []struct {
			Method  string            `json:"method"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"requests"`
	}
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if !got.DryRun || len(got.Requests) != 1 {
		t.Fatalf("出力 = %s", e.Stdout.String())
	}
	if req := got.Requests[0]; req.Method != http.MethodDelete || req.URL != cmdtest.Host+draftPath(12) || req.Headers["If-Match"] != etag {
		t.Errorf("記録した要求 = %+v", req)
	}
	assertNoSecret(t, e)
}

func TestDelete_UsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantMessage string
		wantHint    string
	}{
		{
			name:        "版の指定が無い",
			args:        []string{"12", "--yes"},
			wantMessage: "--if-match",
			wantHint:    "`wikino draft get 12`",
		},
		{
			name:        "--if-matchと--forceの併用",
			args:        []string{"12", "--if-match", etag, "--force"},
			wantMessage: "--if-match, --force",
		},
		{
			name:        "版を問わない--if-match",
			args:        []string{"12", "--if-match", "*"},
			wantMessage: "`*`",
		},
		{
			name:        "不正なページ番号",
			args:        []string{"12?x=1", "--if-match", etag},
			wantMessage: "<page_number>",
		},
		{
			name:        "--dry-runと--fieldsの併用",
			args:        []string{"12", "--if-match", etag, "--dry-run", "--fields", "deleted"},
			wantMessage: "--dry-run, --fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdDraft, append([]string{"draft", "delete"}, tt.args...)...)
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

func TestDelete_APIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		status       int
		wantCode     string
		wantExitCode int
		wantHint     string
	}{
		{name: "412", status: http.StatusPreconditionFailed, wantCode: output.CodePreconditionFailed, wantExitCode: output.ExitConflict, wantHint: "`wikino draft get 12`"},
		{name: "404", status: http.StatusNotFound, wantCode: output.CodeNotFound, wantExitCode: output.ExitNotFound, wantHint: "削除済み"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodDelete, draftPath(12), func(req *http.Request) *http.Response {
				return cmdtest.Problem(tt.status, "失敗しました。")
			})

			err := e.Run(NewCmdDraft, "draft", "delete", "12", "--if-match", etag, "--yes")
			cliErr := cmdtest.AssertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒント %q に %q が無い", cliErr.Hint, tt.wantHint)
			}
		})
	}
}

func TestDelete_DraftNotFoundBeforeConfirmation(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPath(12), func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "下書きが見つかりません。")
	})

	err := e.Run(NewCmdDraft, "draft", "delete", "12", "--if-match", etag)
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "wikino draft list") {
		t.Errorf("ヒントに下書きの一覧のコマンドが無い: %q", cliErr.Hint)
	}
	if e.Stdout.Len() > 0 {
		t.Errorf("標準出力に確認が出た: %q", e.Stdout.String())
	}
}
