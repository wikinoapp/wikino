package page

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const bodyEditsPath = pagePath + "/body_edits"

// handleBodyEditは、ページ12の本文の検索置換の要求を受けて記録し、更新したページを返す。
// ページの取得には版etagのページを、Wikiリンクのプレビューにはリンクの無い結果を返す。
func handleBodyEdit(t *testing.T, e *cmdtest.Env, sent *sentRequest) *previewRecorder {
	t.Helper()

	handleCurrentPage(e, etag)
	e.Handle(http.MethodPost, bodyEditsPath, func(req *http.Request) *http.Response {
		recordRequest(t, req, sent)
		return cmdtest.JSON(http.StatusOK, testPage(12, ptr("議事録"), "# 議事録\n\n- [x] 予算案\n"), "ETag", newETag)
	})

	return handlePreview(t, e)
}

// problemWithErrorsは、`errors` を持つProblem Detailsの応答を返す。
func problemWithErrors(status int, detail string, errors ...map[string]any) *http.Response {
	body, _ := json.Marshal(map[string]any{
		"type": "about:blank", "title": http.StatusText(status), "status": status, "detail": detail, "errors": errors,
	})
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/problem+json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

const editsJSON = `{"edits": [
  {"old_str": "- [ ] 予算案", "new_str": "- [x] 予算案"},
  {"old_str": "TODO", "new_str": "", "replace_all": true},
  {"old_str": "## 関連", "new_str": "## 関連\n\n- [[定例会議]]"}
]}`

func TestUpdate_Edits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		stdin       string
		wantIfMatch string
	}{
		{name: "--if-matchを省く", args: []string{"--edits", writeFile(t, "edits.json", editsJSON)}},
		{name: "--if-matchを付ける", args: []string{"--edits", writeFile(t, "edits.json", editsJSON), "--if-match", etag}, wantIfMatch: etag},
		{name: "標準入力から読む", args: []string{"--edits", "-"}, stdin: editsJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Stdin = tt.stdin
			var sent sentRequest
			preview := handleBodyEdit(t, e, &sent)

			if err := e.Run(NewCmdPage, append([]string{"page", "update", "12"}, tt.args...)...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			// 公開APIと同じ形のJSONを、項目を変えずに送る。
			var want map[string]any
			cmdtest.Decode(t, editsJSON, &want)
			gotJSON, _ := json.Marshal(sent.body)
			wantJSON, _ := json.Marshal(want)
			if !bytes.Equal(gotJSON, wantJSON) {
				t.Errorf("送った編集 = %s、期待値 = %s", gotJSON, wantJSON)
			}
			if sent.contentType != "application/json" {
				t.Errorf("Content-Type = %q、期待値 = application/json", sent.contentType)
			}
			if sent.ifMatch != tt.wantIfMatch {
				t.Errorf("If-Match = %q、期待値 = %q", sent.ifMatch, tt.wantIfMatch)
			}

			// プレビューは空でない各 `new_str` を対象にする。
			wantBodies := []string{"- [x] 予算案", "## 関連\n\n- [[定例会議]]"}
			if got := preview.bodies(); strings.Join(got, "|") != strings.Join(wantBodies, "|") {
				t.Errorf("プレビューに送った本文 = %q、期待値 = %q", got, wantBodies)
			}

			var got map[string]any
			cmdtest.Decode(t, e.Stdout.String(), &got)
			if got["etag"] != newETag || got["number"] != float64(12) {
				t.Errorf("出力 = %v", got)
			}
		})
	}
}

func TestUpdate_EditsWikilinks(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handleCurrentPage(e, etag)
	// 同じリンク先を指す編集が複数あっても、作られるリンク先は1回だけ出す。
	handlePreview(t, e, previewLink("定例会議", "new_page"))

	path := writeFile(t, "edits.json", `{"edits": [{"old_str": "a", "new_str": "[[定例会議]]"}, {"old_str": "b", "new_str": "[[定例会議]]"}]}`)
	if err := e.Run(NewCmdPage, "page", "update", "12", "--edits", path, "--dry-run"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	var got wikilinkDryRunOutput
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if len(got.Requests) != 1 || got.Requests[0].Method != http.MethodPost || got.Requests[0].URL != cmdtest.Host+bodyEditsPath {
		t.Errorf("送る要求 = %+v、期待値 = 検索置換のPOST", got.Requests)
	}
	if len(got.Wikilinks) != 1 || got.Wikilinks[0].PageTitle != "定例会議" {
		t.Errorf("wikilinks = %+v", got.Wikilinks)
	}
	if requests := e.Requests(); len(requests) != 3 {
		t.Errorf("送った要求 = %v、期待値 = 取得とプレビュー2件", methods(requests))
	}
	if !strings.Contains(e.Stderr.String(), "存在しないリンク先1件を未公開のページとして作成します: 開発/定例会議") {
		t.Errorf("標準エラーに作られるリンク先が無い: %q", e.Stderr.String())
	}
}

func TestUpdate_EditsUsageErrors(t *testing.T) {
	t.Parallel()

	valid := writeFile(t, "edits.json", editsJSON)

	tests := []struct {
		name        string
		args        []string
		edits       string
		wantMessage string
	}{
		{name: "JSONでない", edits: "old_str=a", wantMessage: "`--edits` のJSON"},
		{name: "未知の項目", edits: `{"edits": [{"old_string": "a", "new_str": "b"}]}`, wantMessage: "old_string"},
		{name: "new_strが無い", edits: `{"edits": [{"old_str": "a"}]}`, wantMessage: "`edits[0]` に `new_str` がありません"},
		{name: "old_strが無い", edits: `{"edits": [{"old_str": "a", "new_str": "b"}, {"new_str": "c"}]}`, wantMessage: "`edits[1]` に `old_str` がありません"},
		{name: "old_strが空", edits: `{"edits": [{"old_str": "", "new_str": "b"}]}`, wantMessage: "`edits[0].old_str` が空です"},
		{name: "編集が空", edits: `{"edits": []}`, wantMessage: "1件以上"},
		{name: "editsが無い", edits: `{}`, wantMessage: "1件以上"},
		{name: "配列だけ", edits: `[{"old_str": "a", "new_str": "b"}]`, wantMessage: "`--edits` のJSON"},
		{name: "JSONの後の余分なデータ", edits: `{"edits": [{"old_str": "a", "new_str": "b"}]} {}`, wantMessage: "余分なデータ"},
		{name: "UTF-8でない", edits: "{\"edits\": [{\"old_str\": \"\xff\", \"new_str\": \"b\"}]}", wantMessage: "UTF-8"},
		{name: "無いファイル", args: []string{"--edits", filepath.Join(t.TempDir(), "missing.json")}, wantMessage: "missing.json"},
		{name: "--bodyとの併用", args: []string{"--edits", valid, "--body", "本文"}, wantMessage: "--edits, --body"},
		{name: "--body-fileとの併用", args: []string{"--edits", valid, "--body-file", valid}, wantMessage: "--edits, --body-file"},
		{name: "--titleとの併用", args: []string{"--edits", valid, "--title", "議事録"}, wantMessage: "--edits, --title"},
		{name: "--forceとの併用", args: []string{"--edits", valid, "--force"}, wantMessage: "--edits, --force"},
		{name: "版を問わない--if-match", args: []string{"--edits", valid, "--if-match", "*"}, wantMessage: "`--if-match` に `*`"},
		{name: "空の--if-match", args: []string{"--edits", valid, "--if-match", ""}, wantMessage: "`--if-match` の値が空"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := tt.args
			if args == nil {
				args = []string{"--edits", writeFile(t, "edits.json", tt.edits)}
			}

			e := newEnv(t)
			err := e.Run(NewCmdPage, append([]string{"page", "update", "12"}, args...)...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, tt.wantMessage) {
				t.Errorf("メッセージ %q に %q が無い", cliErr.Message, tt.wantMessage)
			}
			assertNoRequest(t, e)
		})
	}
}

func TestUpdate_EditsAPIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		response     func() *http.Response
		wantCode     string
		wantExitCode int
		wantMessage  string
		wantHints    []string
	}{
		{
			name: "一致が複数件",
			response: func() *http.Response {
				return problemWithErrors(http.StatusUnprocessableEntity, "入力の内容に問題があります。",
					map[string]any{"pointer": "#/edits/1/old_str", "detail": "old_strに一致する箇所が本文に2件あります"})
			},
			wantCode:     output.CodeValidationFailed,
			wantExitCode: output.ExitValidation,
			wantMessage:  "2件",
			wantHints:    []string{"2番目の編集 (`edits[1]`)", "`wikino page get 12 --md`", "どの編集も適用していません"},
		},
		{
			name: "一致が0件",
			response: func() *http.Response {
				return problemWithErrors(http.StatusUnprocessableEntity, "入力の内容に問題があります。",
					map[string]any{"pointer": "#/edits/0/old_str", "detail": "old_strに一致する箇所が本文にありません"})
			},
			wantCode:     output.CodeValidationFailed,
			wantExitCode: output.ExitValidation,
			wantMessage:  "一致する箇所が本文にありません",
			wantHints:    []string{"1番目の編集 (`edits[0]`)", "`wikino page get 12 --md`"},
		},
		{
			name: "本文の上限",
			response: func() *http.Response {
				return problemWithErrors(http.StatusUnprocessableEntity, "入力の内容に問題があります。",
					map[string]any{"pointer": "#/edits/2/new_str", "detail": "本文が長すぎます"})
			},
			wantCode:     output.CodeValidationFailed,
			wantExitCode: output.ExitValidation,
			wantHints:    []string{"3番目の編集 (`edits[2]`)", "上限"},
		},
		{
			name: "編集でない項目の422",
			response: func() *http.Response {
				return problemWithErrors(http.StatusUnprocessableEntity, "入力の内容に問題があります。",
					map[string]any{"pointer": "#/title", "detail": "タイトルを入力してください"})
			},
			wantCode:     output.CodeValidationFailed,
			wantExitCode: output.ExitValidation,
			wantHints:    []string{"--title"},
		},
		{
			name:         "読んだ後の更新",
			response:     func() *http.Response { return cmdtest.Problem(http.StatusPreconditionFailed, "失敗しました。") },
			wantCode:     output.CodePreconditionFailed,
			wantExitCode: output.ExitConflict,
			wantHints:    []string{"`wikino page get 12 --md`", "`old_str`"},
		},
		{
			name:         "ページが無い",
			response:     func() *http.Response { return cmdtest.Problem(http.StatusNotFound, "失敗しました。") },
			wantCode:     output.CodeNotFound,
			wantExitCode: output.ExitNotFound,
			wantHints:    []string{"wikino page list"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			handleCurrentPage(e, etag)
			handlePreview(t, e)
			e.Handle(http.MethodPost, bodyEditsPath, func(req *http.Request) *http.Response {
				return tt.response()
			})

			err := e.Run(NewCmdPage, "page", "update", "12", "--edits", writeFile(t, "edits.json", editsJSON), "--if-match", etag)
			cliErr := cmdtest.AssertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Message, tt.wantMessage) {
				t.Errorf("メッセージ %q に %q が無い", cliErr.Message, tt.wantMessage)
			}
			for _, want := range tt.wantHints {
				if !strings.Contains(cliErr.Hint, want) {
					t.Errorf("ヒント %q に %q が無い", cliErr.Hint, want)
				}
			}
		})
	}
}
