package page

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// sentRequestは、APIのスタブが受けた書き込みの要求の中身。
type sentRequest struct {
	contentType string
	ifMatch     string
	body        map[string]any
}

// handleCreateは、ページの作成の要求を受けて記録し、作成したページを返す。
// Wikiリンクのプレビューには、リンクの無い結果を返す。
func handleCreate(t *testing.T, e *cmdtest.Env, sent *sentRequest) {
	t.Helper()

	handlePreview(t, e)
	e.Handle(http.MethodPost, pagesPath, func(req *http.Request) *http.Response {
		recordRequest(t, req, sent)
		title, _ := sent.body["title"].(string)
		body, _ := sent.body["body"].(string)
		return cmdtest.JSON(http.StatusCreated, testPage(12, &title, body), "ETag", etag, "Location", "https://wikino.example.com/s/team/pages/12")
	})
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
	cmdtest.Decode(t, string(data), &sent.body)
}

func TestCreate_JSON(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	var sent sentRequest
	handleCreate(t, e, &sent)

	if err := e.Run(NewCmdPage, "page", "create", "--topic", "3", "--title", "議事録", "--body", "# 議事録\n"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	if sent.contentType != "application/json" {
		t.Errorf("Content-Type = %q、期待値 = application/json", sent.contentType)
	}
	want := map[string]any{"topic_number": float64(3), "title": "議事録", "body": "# 議事録\n"}
	for key, value := range want {
		if sent.body[key] != value {
			t.Errorf("送った %s = %v、期待値 = %v", key, sent.body[key], value)
		}
	}

	var got map[string]any
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if got["etag"] != etag || got["number"] != float64(12) || got["body"] != "# 議事録\n" {
		t.Errorf("出力 = %v", got)
	}
	if e.Stderr.Len() > 0 {
		t.Errorf("JSONの出力で標準エラーに出力がある: %q", e.Stderr.String())
	}
}

func TestCreate_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	var sent sentRequest
	handleCreate(t, e, &sent)

	if err := e.RunTTY(NewCmdPage, "page", "create", "--topic", "3", "--title", "議事録", "--body", "# 議事録\n\n本文\n"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "create", e.Stdout.String())
	if got, want := e.Stderr.String(), "ページ12を作成しました。\n"; got != want {
		t.Errorf("標準エラー = %q、期待値 = %q", got, want)
	}
}

func TestCreate_BodyInput(t *testing.T) {
	t.Parallel()

	// 末尾の改行や空白も本文の一部として、削らずに送る。
	const body = "# 議事録\n\n- [ ] 予算案\n\n"
	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{name: "ファイル", args: []string{"--body-file", path}, want: body},
		{name: "標準入力", args: []string{"--body-file", "-"}, stdin: body, want: body},
		{name: "空の本文", args: []string{"--body", ""}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Stdin = tt.stdin
			var sent sentRequest
			handleCreate(t, e, &sent)

			args := append([]string{"page", "create", "--topic", "3", "--title", "議事録"}, tt.args...)
			if err := e.Run(NewCmdPage, args...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if got, ok := sent.body["body"].(string); !ok || got != tt.want {
				t.Errorf("送った本文 = %q、期待値 = %q", sent.body["body"], tt.want)
			}
		})
	}
}

func TestCreate_UsageErrors(t *testing.T) {
	t.Parallel()

	invalidUTF8 := filepath.Join(t.TempDir(), "invalid.md")
	if err := os.WriteFile(invalidUTF8, []byte{0xff, 0xfe, 'a'}, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{name: "トピックが無い", args: []string{"--title", "議事録", "--body", ""}, wantMessage: "--topic"},
		{name: "タイトルが無い", args: []string{"--topic", "3", "--body", ""}, wantMessage: "--title"},
		{name: "本文が無い", args: []string{"--topic", "3", "--title", "議事録"}, wantMessage: "--body ''"},
		{name: "本文のフラグの併用", args: []string{"--topic", "3", "--title", "議事録", "--body", "a", "--body-file", "-"}, wantMessage: "--body, --body-file"},
		{name: "不正なトピック番号", args: []string{"--topic", "0", "--title", "議事録", "--body", ""}, wantMessage: "--topic"},
		{name: "無いファイル", args: []string{"--topic", "3", "--title", "議事録", "--body-file", filepath.Join(t.TempDir(), "missing.md")}, wantMessage: "missing.md"},
		{name: "UTF-8でない本文", args: []string{"--topic", "3", "--title", "議事録", "--body-file", invalidUTF8}, wantMessage: "UTF-8"},
		{name: "位置引数", args: []string{"12", "--topic", "3", "--title", "議事録", "--body", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdPage, append([]string{"page", "create"}, tt.args...)...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, tt.wantMessage) {
				t.Errorf("メッセージ %q に %q が無い", cliErr.Message, tt.wantMessage)
			}
			assertNoRequest(t, e)
		})
	}
}

func TestCreate_DryRun(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePreview(t, e)
	if err := e.Run(NewCmdPage, "page", "create", "--topic", "3", "--title", "議事録", "--body", "# 議事録\n", "--dry-run"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	// Wikiリンクのプレビュー (読み取り) だけを送り、作成の要求は送らない。
	if requests := e.Requests(); len(requests) != 1 || requests[0].URL.Path != previewPath {
		t.Errorf("送った要求 = %v、期待値 = プレビューの1件", methods(requests))
	}

	got := decodeDryRun(t, e.Stdout.String())
	if len(got.Requests) != 1 {
		t.Fatalf("記録した要求 = %d 件、期待値 = 1 件: %s", len(got.Requests), e.Stdout.String())
	}
	req := got.Requests[0]
	if req.Method != http.MethodPost || req.URL != cmdtest.Host+pagesPath {
		t.Errorf("要求 = %s %s", req.Method, req.URL)
	}
	if req.Body["title"] != "議事録" || req.Body["body"] != "# 議事録\n" || req.Body["topic_number"] != float64(3) {
		t.Errorf("要求の本文 = %v", req.Body)
	}
	assertNoSecret(t, e)
	if !strings.Contains(e.Stderr.String(), "--dry-run") {
		t.Errorf("標準エラーに要求を送っていないことの案内が無い: %q", e.Stderr.String())
	}
}

func TestCreate_APIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		status       int
		wantCode     string
		wantExitCode int
		wantHint     string
	}{
		{name: "422", status: http.StatusUnprocessableEntity, wantCode: output.CodeValidationFailed, wantExitCode: output.ExitValidation, wantHint: "`wikino page list --topic 3 --title '議事録 (1)'`"},
		{name: "404", status: http.StatusNotFound, wantCode: output.CodeNotFound, wantExitCode: output.ExitNotFound, wantHint: "トピック3"},
		{name: "5xxは作成済みかを確かめさせる", status: http.StatusInternalServerError, wantCode: output.CodeServerError, wantExitCode: output.ExitError, wantHint: "`wikino page list --topic 3 --title '議事録 (1)'`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			handlePreview(t, e)
			e.Handle(http.MethodPost, pagesPath, func(req *http.Request) *http.Response {
				return cmdtest.Problem(tt.status, "失敗しました。")
			})

			err := e.Run(NewCmdPage, "page", "create", "--topic", "3", "--title", "議事録 (1)", "--body", "")
			cliErr := cmdtest.AssertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒント %q に %q が無い", cliErr.Hint, tt.wantHint)
			}
		})
	}
}

func TestCreate_DryRunWithFields(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	err := e.Run(NewCmdPage, "page", "create", "--topic", "3", "--title", "議事録", "--body", "本文", "--dry-run", "--fields", "number,etag")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
	if !strings.Contains(cliErr.Message, "--dry-run, --fields") || !strings.Contains(cliErr.Hint, "`--fields` を外して") {
		t.Errorf("併用できないフラグと外すフラグの案内が無い: %v", cliErr)
	}
	assertNoRequest(t, e)
	if e.Stdout.Len() > 0 {
		t.Errorf("拒否時の標準出力に結果がある: %q", e.Stdout.String())
	}
}
