package draft

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const (
	// newETagは、更新後の下書きの版。
	newETag       = `"9e107d9d"`
	bodyEditsPath = pagesPath + "/12/draft_page/body_edits"
)

// sentRequestは、書き込みの要求で送った内容。
type sentRequest struct {
	contentType string
	ifMatch     string
	body        map[string]any
}

func recordRequest(t *testing.T, req *http.Request, sent *sentRequest) {
	t.Helper()

	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("要求の本文を読めない: %v", err)
	}
	sent.contentType = req.Header.Get("Content-Type")
	sent.ifMatch = req.Header.Get("If-Match")
	sent.body = map[string]any{}
	if len(data) > 0 {
		cmdtest.Decode(t, string(data), &sent.body)
	}
}

// handleCurrentDraftは、ページ12の下書きの取得の要求に、トピック3の下書きと版currentETagで応答する。
func handleCurrentDraft(e *cmdtest.Env, currentETag string) {
	e.Handle(http.MethodGet, draftPath(12), func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testDraft(12, ptr("議事録"), "# 議事録\n"), "ETag", currentETag)
	})
}

// handleUpdateは、ページ12の下書きの更新の要求を受けて記録し、更新した下書きを返す。
// 下書きの取得には版etagの下書きを、Wikiリンクのプレビューにはリンクの無い結果を返す。
func handleUpdate(t *testing.T, e *cmdtest.Env, sent *sentRequest) *previewRecorder {
	t.Helper()

	handleCurrentDraft(e, etag)
	e.Handle(http.MethodPatch, draftPath(12), func(req *http.Request) *http.Response {
		recordRequest(t, req, sent)
		draft := testDraft(12, ptr("議事録"), "# 議事録\n")
		for key, value := range sent.body {
			draft[key] = value
		}
		return cmdtest.JSON(http.StatusOK, draft, "ETag", newETag)
	})

	return handlePreview(t, e)
}

func TestUpdate_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantBody map[string]any
		// wantPreviewは、本文を書き込むためにプレビューを送るか。
		wantPreview bool
	}{
		{
			name:        "本文だけ",
			args:        []string{"--body", "# 議事録\n\n追記\n"},
			wantBody:    map[string]any{"body": "# 議事録\n\n追記\n"},
			wantPreview: true,
		},
		{
			name:     "タイトルだけ",
			args:     []string{"--title", "定例会議の議事録"},
			wantBody: map[string]any{"title": "定例会議の議事録"},
		},
		{
			name:        "タイトルと本文",
			args:        []string{"--title", "定例会議の議事録", "--body", ""},
			wantBody:    map[string]any{"title": "定例会議の議事録", "body": ""},
			wantPreview: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			var sent sentRequest
			preview := handleUpdate(t, e, &sent)

			args := append([]string{"draft", "update", "12", "--if-match", etag}, tt.args...)
			if err := e.Run(NewCmdDraft, args...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			if sent.ifMatch != etag {
				t.Errorf("If-Match = %q、期待値 = %q", sent.ifMatch, etag)
			}
			if sent.contentType != "application/merge-patch+json" {
				t.Errorf("Content-Type = %q、期待値 = application/merge-patch+json", sent.contentType)
			}
			// 指定しなかった項目は送らず、変えない。
			gotJSON, _ := json.Marshal(sent.body)
			wantJSON, _ := json.Marshal(tt.wantBody)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("送った本文 = %s、期待値 = %s", gotJSON, wantJSON)
			}
			// 空の本文はリンクを持たないため、プレビューを送らない。
			if got := len(preview.sent()) > 0; got != tt.wantPreview {
				t.Errorf("プレビューを送った = %v、期待値 = %v", got, tt.wantPreview)
			}

			var got map[string]any
			cmdtest.Decode(t, e.Stdout.String(), &got)
			if got["etag"] != newETag || got["page_number"] != float64(12) {
				t.Errorf("出力 = %v、期待値 = 更新後の版 %s のページ12の下書き", got, newETag)
			}
		})
	}
}

func TestUpdate_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	var sent sentRequest
	handleUpdate(t, e, &sent)

	if err := e.RunTTY(NewCmdDraft, "draft", "update", "12", "--if-match", etag, "--body", "# 議事録\n\n追記\n"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "update", e.Stdout.String())
	if got, want := e.Stderr.String(), "ページ12の下書きを更新しました。\n"; got != want {
		t.Errorf("標準エラー = %q、期待値 = %q", got, want)
	}
}

func TestUpdate_PreviewUsesDraftTopic(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	var sent sentRequest
	handleUpdate(t, e, &sent)
	handlePreview(t, e, newPageLink("定例会議"))

	if err := e.Run(NewCmdDraft, "draft", "update", "12", "--if-match", etag, "--body", "[[定例会議]]\n"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// プレビューに要るトピックの番号は、ページではなく下書きの取得から得る (draft_page:writeで足りる)。
	want := []string{"GET " + draftPath(12), "POST " + previewPath, "PATCH " + draftPath(12)}
	if got := requestPaths(e); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("要求の順 = %v、期待値 = %v", got, want)
	}
	if !strings.Contains(e.Stderr.String(), "開発/定例会議") {
		t.Errorf("標準エラーに作られたリンク先が無い: %q", e.Stderr.String())
	}
}

func TestUpdate_Force(t *testing.T) {
	t.Parallel()

	const latestETag = `"latest"`
	e := newEnv(t)
	var sent sentRequest
	handleUpdate(t, e, &sent)
	handleCurrentDraft(e, latestETag)

	if err := e.Run(NewCmdDraft, "draft", "update", "12", "--force", "--title", "上書き"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// タイトルだけの更新でも、`--force` では最新の版を取得してからその版で更新する。
	want := []string{"GET " + draftPath(12), "PATCH " + draftPath(12)}
	if got := requestPaths(e); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("要求の順 = %v、期待値 = %v", got, want)
	}
	if sent.ifMatch != latestETag {
		t.Errorf("If-Match = %q、期待値 = %q (取得した最新の版)", sent.ifMatch, latestETag)
	}
}

// writeFileは、一時ディレクトリにcontentのファイルを作ってパスを返す。
func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

// `draft get --md` で書き出したファイルを書き戻すと、下書きのAPIに本文・タイトル・版がそのまま送られる。
func TestUpdate_MarkdownFile(t *testing.T) {
	t.Parallel()

	const body = "---\ntags: [議事録]   # 分類\n---\n\n本文\n"
	export := newEnv(t)
	export.Handle(http.MethodGet, draftPath(12), func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testDraft(12, ptr("議事録"), body), "ETag", etag)
	})
	if err := export.Run(NewCmdDraft, "draft", "get", "12", "--md"); err != nil {
		t.Fatalf("書き出しでエラーを期待しなかったが %v だった", err)
	}
	path := writeFile(t, "draft.md", export.Stdout.String())

	e := newEnv(t)
	var sent sentRequest
	handleUpdate(t, e, &sent)
	if err := e.Run(NewCmdDraft, "draft", "update", "12", "--md-file", path); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	if got := sent.body["body"]; got != body {
		t.Errorf("送った本文 = %q、期待値 = %q", got, body)
	}
	if got := sent.body["title"]; got != "議事録" {
		t.Errorf("送ったタイトル = %v、期待値 = 議事録", got)
	}
	if sent.ifMatch != etag {
		t.Errorf("If-Match = %q、期待値 = %q (ファイルのwikino_etag)", sent.ifMatch, etag)
	}
}

func TestUpdate_MarkdownFileWithoutTitle(t *testing.T) {
	t.Parallel()

	// タイトルの無い下書きの書き出しには `wikino_title` が無く、書き戻してもタイトルを送らない。
	path := writeFile(t, "draft.md", "---\nwikino_number: 12\nwikino_etag: '"+etag+"'\n---\n\n本文\n")

	e := newEnv(t)
	var sent sentRequest
	handleUpdate(t, e, &sent)
	if err := e.Run(NewCmdDraft, "draft", "update", "12", "--md-file", path); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	if _, ok := sent.body["title"]; ok {
		t.Errorf("タイトルを送らないことを期待したが %v を送った", sent.body["title"])
	}
	if got := sent.body["body"]; got != "本文\n" {
		t.Errorf("送った本文 = %q、期待値 = %q", got, "本文\n")
	}
}

func TestUpdate_MarkdownFilePreconditionFailed(t *testing.T) {
	t.Parallel()

	path := writeFile(t, "draft.md", "---\nwikino_number: 12\nwikino_etag: '"+etag+"'\n---\n\n本文\n")

	e := newEnv(t)
	handleCurrentDraft(e, etag)
	handlePreview(t, e)
	e.Handle(http.MethodPatch, draftPath(12), func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusPreconditionFailed, "下書きはその後に更新されています。")
	})

	err := e.Run(NewCmdDraft, "draft", "update", "12", "--md-file", path)
	cliErr := cmdtest.AssertCLIError(t, err, output.CodePreconditionFailed, output.ExitConflict)
	if !strings.Contains(cliErr.Hint, "`wikino draft get 12 --md > draft.md`") {
		t.Errorf("ヒントに書き出し直すコマンドが無い: %q", cliErr.Hint)
	}
}

func TestUpdate_Edits(t *testing.T) {
	t.Parallel()

	const editsJSON = `{"edits": [{"old_str": "- [ ] 予算案", "new_str": "- [x] 予算案 [[予算]]"}]}`

	tests := []struct {
		name        string
		args        []string
		wantIfMatch string
	}{
		{name: "--if-matchを省く", args: []string{"--edits", "-"}},
		{name: "--if-matchを付ける", args: []string{"--edits", "-", "--if-match", etag}, wantIfMatch: etag},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Stdin = editsJSON
			handleCurrentDraft(e, etag)
			preview := handlePreview(t, e)
			var sent sentRequest
			e.Handle(http.MethodPost, bodyEditsPath, func(req *http.Request) *http.Response {
				recordRequest(t, req, &sent)
				return cmdtest.JSON(http.StatusOK, testDraft(12, ptr("議事録"), "- [x] 予算案 [[予算]]\n"), "ETag", newETag)
			})

			if err := e.Run(NewCmdDraft, append([]string{"draft", "update", "12"}, tt.args...)...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			// ページではなく下書きの検索置換のAPIを呼ぶ。
			want := []string{"GET " + draftPath(12), "POST " + previewPath, "POST " + bodyEditsPath}
			if got := requestPaths(e); strings.Join(got, "|") != strings.Join(want, "|") {
				t.Errorf("要求の順 = %v、期待値 = %v", got, want)
			}
			edits, _ := sent.body["edits"].([]any)
			if len(edits) != 1 {
				t.Errorf("送った編集 = %v", sent.body)
			}
			if sent.ifMatch != tt.wantIfMatch {
				t.Errorf("If-Match = %q、期待値 = %q", sent.ifMatch, tt.wantIfMatch)
			}
			if got := preview.sent(); len(got) != 1 || got[0]["body"] != "- [x] 予算案 [[予算]]" {
				t.Errorf("プレビューに送った本文 = %v、期待値 = new_str", got)
			}

			var got map[string]any
			cmdtest.Decode(t, e.Stdout.String(), &got)
			if got["etag"] != newETag {
				t.Errorf("出力のetag = %v、期待値 = %s", got["etag"], newETag)
			}
		})
	}
}

func TestUpdate_EditsNotMatched(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Stdin = `{"edits": [{"old_str": "a", "new_str": "b"}, {"old_str": "c", "new_str": "d"}]}`
	handleCurrentDraft(e, etag)
	handlePreview(t, e)
	e.Handle(http.MethodPost, bodyEditsPath, func(req *http.Request) *http.Response {
		body, _ := json.Marshal(map[string]any{
			"type": "about:blank", "title": "Unprocessable Entity", "status": 422, "detail": "入力が正しくありません。",
			"errors": []map[string]any{{"pointer": "#/edits/1/old_str", "detail": "old_strに一致する箇所がありません (0件)。"}},
		})
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Header:     http.Header{"Content-Type": {"application/problem+json"}},
			Body:       io.NopCloser(bytes.NewReader(body)),
		}
	})

	err := e.Run(NewCmdDraft, "draft", "update", "12", "--edits", "-")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeValidationFailed, output.ExitValidation)
	if !strings.Contains(cliErr.Hint, "2番目") || !strings.Contains(cliErr.Hint, "`wikino draft get 12 --md`") {
		t.Errorf("ヒントに何番目の編集かと今の本文を確かめるコマンドが無い: %q", cliErr.Hint)
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
			wantHint:    "`wikino draft get 12`",
		},
		{
			name:        "--if-matchと--forceの併用",
			args:        []string{"12", "--body", "本文", "--if-match", etag, "--force"},
			wantMessage: "--if-match, --force",
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
			name:        "--md-fileと--bodyの併用",
			args:        []string{"12", "--md-file", "draft.md", "--body", "本文"},
			wantMessage: "--md-file, --body",
		},
		{
			name:        "--editsと--titleの併用",
			args:        []string{"12", "--edits", "-", "--title", "題"},
			wantMessage: "--title",
		},
		{
			name:        "不正なページ番号",
			args:        []string{"../12", "--if-match", etag, "--body", "本文"},
			wantMessage: "<page_number>",
		},
		{
			name:        "--dry-runと--fieldsの併用",
			args:        []string{"12", "--if-match", etag, "--body", "本文", "--dry-run", "--fields", "etag"},
			wantMessage: "--dry-run, --fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdDraft, append([]string{"draft", "update"}, tt.args...)...)
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

	e := newEnv(t)
	handleCurrentDraft(e, etag)
	handlePreview(t, e, newPageLink("定例会議"))

	if err := e.Run(NewCmdDraft, "draft", "update", "12", "--if-match", etag, "--body", "[[定例会議]]\n", "--dry-run"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// 下書きの取得とWikiリンクのプレビュー (読み取り) だけを送り、更新の要求は送らない。
	want := []string{"GET " + draftPath(12), "POST " + previewPath}
	if got := requestPaths(e); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("送った要求 = %v、期待値 = %v", got, want)
	}

	var got struct {
		DryRun   bool `json:"dry_run"`
		Requests []struct {
			Method  string            `json:"method"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"requests"`
		Wikilinks []map[string]any `json:"wikilinks"`
	}
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if !got.DryRun || len(got.Requests) != 1 {
		t.Fatalf("出力 = %s", e.Stdout.String())
	}
	if req := got.Requests[0]; req.Method != http.MethodPatch || req.URL != cmdtest.Host+draftPath(12) || req.Headers["If-Match"] != etag {
		t.Errorf("記録した要求 = %+v", req)
	}
	if len(got.Wikilinks) != 1 || got.Wikilinks[0]["resolution"] != "new_page" {
		t.Errorf("wikilinks = %v", got.Wikilinks)
	}
	assertNoSecret(t, e)
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
		{name: "412", status: http.StatusPreconditionFailed, wantCode: output.CodePreconditionFailed, wantExitCode: output.ExitConflict, wantHint: "`wikino draft get 12`"},
		{name: "428", status: http.StatusPreconditionRequired, wantCode: output.CodePreconditionRequired, wantExitCode: output.ExitConflict, wantHint: "`wikino draft get 12`"},
		{name: "422", status: http.StatusUnprocessableEntity, wantCode: output.CodeValidationFailed, wantExitCode: output.ExitValidation, wantHint: "メッセージに示した項目"},
		{name: "404", status: http.StatusNotFound, wantCode: output.CodeNotFound, wantExitCode: output.ExitNotFound, wantHint: "wikino draft create"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodPatch, draftPath(12), func(req *http.Request) *http.Response {
				return cmdtest.Problem(tt.status, "失敗しました。")
			})

			err := e.Run(NewCmdDraft, "draft", "update", "12", "--if-match", etag, "--title", "題")
			cliErr := cmdtest.AssertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒント %q に %q が無い", cliErr.Hint, tt.wantHint)
			}
		})
	}
}

func TestUpdate_DraftNotFoundBeforePreview(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPath(12), func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "下書きが見つかりません。")
	})

	err := e.Run(NewCmdDraft, "draft", "update", "12", "--if-match", etag, "--body", "本文")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "wikino draft create") {
		t.Errorf("ヒントに下書きの作成のコマンドが無い: %q", cliErr.Hint)
	}
	if got := requestPaths(e); len(got) != 1 {
		t.Errorf("要求 = %v、期待値 = 取得の1件だけ", got)
	}
}
