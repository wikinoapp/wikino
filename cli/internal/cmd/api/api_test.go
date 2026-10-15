package api

import (
	"bytes"
	"errors"
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
	pagesPath     = "/api/v1/spaces/team/pages"
	draftPagePath = "/api/v1/spaces/team/pages/12/draft_page"
	etag          = `"d41d8cd9"`
)

// pagesBodyは、一覧の応答の本文。APIが返したバイト列のまま出すことを確かめるため、キーの順をあえて崩す。
const pagesBody = `{"next_cursor":null,"items":[{"title":"議事録","number":12}]}` + "\n"

// rawResponseは、bodyをそのまま本文とするstatusの応答を返す。
func rawResponse(status int, contentType, body string) *http.Response {
	h := http.Header{}
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}

	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body))}
}

// handleRecordは、methodとpathの要求にresponseを返し、受けた要求の本文を記録する。
func handleRecord(e *cmdtest.Env, method, path string, response func() *http.Response) *[]string {
	var bodies []string
	e.Handle(method, path, func(req *http.Request) *http.Response {
		var body []byte
		if req.Body != nil {
			body, _ = io.ReadAll(req.Body)
		}
		bodies = append(bodies, string(body))
		return response()
	})

	return &bodies
}

func assertNoRequest(t *testing.T, e *cmdtest.Env) {
	t.Helper()

	if requests := e.Requests(); len(requests) > 0 {
		t.Errorf("要求を送らないことを期待したが %d 件送った: %s %s", len(requests), requests[0].Method, requests[0].URL)
	}
}

func TestAPI_Get(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleRecord(e, http.MethodGet, pagesPath, func() *http.Response {
		return rawResponse(http.StatusOK, "application/json", pagesBody)
	})

	// メソッドは大文字と小文字を区別しない。パスの `{space}` は実行するスペースに置き換える。
	if err := e.Run(NewCmdAPI, "api", "get", "/spaces/{space}/pages?limit=5"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	if got := e.Stdout.String(); got != pagesBody {
		t.Errorf("標準出力 = %q、期待値 = %q", got, pagesBody)
	}
	req := e.Requests()[0]
	if req.Method != http.MethodGet || req.URL.RawQuery != "limit=5" {
		t.Errorf("要求 = %s %s、期待値 = GET %s?limit=5", req.Method, req.URL, pagesPath)
	}
	if got := req.Header.Get("Content-Type"); got != "" {
		t.Errorf("本文の無い要求にContent-Typeが付いている: %q", got)
	}
}

func TestAPI_GetTTY(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleRecord(e, http.MethodGet, pagesPath, func() *http.Response {
		return rawResponse(http.StatusOK, "application/json", strings.TrimSuffix(pagesBody, "\n"))
	})

	if err := e.RunTTY(NewCmdAPI, "api", "GET", "/spaces/team/pages"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// TTYでは字下げするが、キーの順は応答のまま保つ。
	want := `{
  "next_cursor": null,
  "items": [
    {
      "title": "議事録",
      "number": 12
    }
  ]
}
`
	if got := e.Stdout.String(); got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestAPI_NotJSON(t *testing.T) {
	t.Parallel()

	const yaml = "openapi: 3.1.0\ninfo:\n  title: Wikino\n"
	e := cmdtest.New(t)
	handleRecord(e, http.MethodGet, "/api/v1/openapi.yaml", func() *http.Response {
		return rawResponse(http.StatusOK, "application/yaml", yaml)
	})

	if err := e.RunTTY(NewCmdAPI, "api", "GET", "/openapi.yaml"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if got := e.Stdout.String(); got != yaml {
		t.Errorf("標準出力 = %q、期待値 = %q", got, yaml)
	}

	// JSONでない応答には `--jq` を適用できない。要求は送っているため、使い方の誤りにはしない。
	err := e.Run(NewCmdAPI, "api", "GET", "/openapi.yaml", "--jq", ".")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeJQFailed, output.ExitError)
	if !strings.Contains(cliErr.Message, "application/yaml") {
		t.Errorf("メッセージに応答の種類が無い: %q", cliErr.Message)
	}
}

func TestAPI_JQ(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleRecord(e, http.MethodGet, pagesPath, func() *http.Response {
		return rawResponse(http.StatusOK, "application/json", pagesBody)
	})

	if err := e.Run(NewCmdAPI, "api", "GET", "/spaces/team/pages", "--jq", ".items[0].title"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if got, want := e.Stdout.String(), "議事録\n"; got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestAPI_PatchWithInputFile(t *testing.T) {
	t.Parallel()

	const body = `{"title": "定例会議"}`
	path := filepath.Join(t.TempDir(), "page.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	e := cmdtest.New(t)
	bodies := handleRecord(e, http.MethodPatch, pagesPath+"/12", func() *http.Response {
		return rawResponse(http.StatusOK, "application/json", `{"number":12}`)
	})

	if err := e.Run(NewCmdAPI, "api", "PATCH", "/spaces/{space}/pages/12", "--input", path, "--header", "If-Match: "+etag); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	req := e.Requests()[0]
	if got := req.Header.Get("If-Match"); got != etag {
		t.Errorf("If-Match = %q、期待値 = %q", got, etag)
	}
	// 公開APIのPATCHはJSON Merge Patchだけを受け付ける。
	if got, want := req.Header.Get("Content-Type"), "application/merge-patch+json"; got != want {
		t.Errorf("Content-Type = %q、期待値 = %q", got, want)
	}
	if len(*bodies) != 1 || (*bodies)[0] != body {
		t.Errorf("本文 = %q、期待値 = %q", *bodies, body)
	}
	if got, want := e.Stdout.String(), `{"number":12}`; got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestAPI_PostWithStdin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		args            []string
		wantContentType string
	}{
		{name: "既定はJSON", wantContentType: "application/json"},
		{name: "Content-Typeを指定する", args: []string{"-H", "content-type: application/json; charset=utf-8"}, wantContentType: "application/json; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// 末尾の改行も本文として送る。
			const body = "{\"topic_number\": 3, \"title\": \"議事録\", \"body\": \"本文\\n\"}\n"
			e := cmdtest.New(t)
			e.Stdin = body
			bodies := handleRecord(e, http.MethodPost, pagesPath, func() *http.Response {
				return rawResponse(http.StatusCreated, "application/json", `{"number":13}`)
			})

			args := append([]string{"api", "POST", "spaces/team/pages", "--input", "-"}, tt.args...)
			if err := e.Run(NewCmdAPI, args...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			if got := e.Requests()[0].Header.Get("Content-Type"); got != tt.wantContentType {
				t.Errorf("Content-Type = %q、期待値 = %q", got, tt.wantContentType)
			}
			if len(*bodies) != 1 || (*bodies)[0] != body {
				t.Errorf("本文 = %q、期待値 = %q", *bodies, body)
			}
		})
	}
}

func TestAPI_NoContent(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleRecord(e, http.MethodDelete, draftPagePath, func() *http.Response {
		return rawResponse(http.StatusNoContent, "", "")
	})

	if err := e.RunTTY(NewCmdAPI, "api", "DELETE", "/spaces/team/pages/12/draft_page", "-H", "If-Match: "+etag, "--yes"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if got := e.Stdout.String(); got != "" {
		t.Errorf("本文の無い応答で標準出力に出した: %q", got)
	}
}

func TestAPI_ErrorStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status       int
		wantCode     string
		wantExitCode int
	}{
		{http.StatusBadRequest, output.CodeBadRequest, output.ExitValidation},
		{http.StatusUnauthorized, output.CodeUnauthorized, output.ExitAuth},
		{http.StatusForbidden, output.CodeForbidden, output.ExitForbidden},
		{http.StatusNotFound, output.CodeNotFound, output.ExitNotFound},
		{http.StatusPreconditionFailed, output.CodePreconditionFailed, output.ExitConflict},
		{http.StatusPreconditionRequired, output.CodePreconditionRequired, output.ExitConflict},
		{http.StatusUnprocessableEntity, output.CodeValidationFailed, output.ExitValidation},
		{http.StatusTooManyRequests, output.CodeRateLimited, output.ExitRateLimited},
		{http.StatusInternalServerError, output.CodeServerError, output.ExitError},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			t.Parallel()

			e := cmdtest.New(t)
			handleRecord(e, http.MethodPatch, pagesPath+"/12", func() *http.Response {
				return cmdtest.Problem(tt.status, "APIの説明")
			})
			e.Stdin = `{"title": "a"}`

			err := e.Run(NewCmdAPI, "api", "PATCH", "/spaces/team/pages/12", "--input", "-")
			cliErr := cmdtest.AssertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if cliErr.Status != tt.status || cliErr.Message != "APIの説明" {
				t.Errorf("ステータス・メッセージ = %d, %q、期待値 = %d, %q", cliErr.Status, cliErr.Message, tt.status, "APIの説明")
			}
			if got := e.Stdout.String(); got != "" {
				t.Errorf("失敗の応答で標準出力に出した: %q", got)
			}
		})
	}
}

func TestAPI_UsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		// wantMessageは、メッセージに含まれるべき文字列。
		wantMessage string
	}{
		{name: "引数が足りない", args: []string{"GET"}, wantMessage: "引数"},
		{name: "使えないメソッド", args: []string{"HEAD", "/spaces/team/pages"}, wantMessage: `"HEAD"`},
		{name: "URL", args: []string{"GET", "https://evil.example.com/api/v1/spaces/team/pages"}, wantMessage: "/api/v1` からのパス"},
		{name: "スキームの無いホスト", args: []string{"GET", "//evil.example.com/spaces/team/pages"}, wantMessage: "/api/v1` からのパス"},
		{name: "基底のパスを重ねる", args: []string{"GET", "/api/v1/spaces/team/pages"}, wantMessage: "/api/v1` からのパス"},
		{name: "上の階層", args: []string{"GET", "/spaces/team/../../../oauth/token"}, wantMessage: "`..`"},
		{name: "エンコードした上の階層", args: []string{"GET", "/%2e%2e/oauth/token"}, wantMessage: "`..`"},
		{name: "フラグメント", args: []string{"GET", "/spaces/team/pages#x"}, wantMessage: "/api/v1` からのパス"},
		{name: "コロンの無いヘッダー", args: []string{"GET", "/spaces/team/pages", "-H", "X-Test"}, wantMessage: "--header"},
		{name: "改行を含むヘッダー", args: []string{"GET", "/spaces/team/pages", "-H", "X-Test: a\r\nX-Other: b"}, wantMessage: "--header"},
		{name: "空白を含むヘッダーの名前", args: []string{"GET", "/spaces/team/pages", "-H", "X Test: a"}, wantMessage: "--header"},
		{name: "2番目の誤ったヘッダー", args: []string{"GET", "/spaces/team/pages", "-H", "X-Test: a", "-H", "X-Other"}, wantMessage: "2番目の `--header`"},
		{name: "Authorizationヘッダー", args: []string{"GET", "/spaces/team/pages", "-H", "authorization: Bearer other"}, wantMessage: "Authorization"},
		{name: "Cookieヘッダー", args: []string{"GET", "/spaces/team/pages", "-H", "Cookie: session=x"}, wantMessage: "Cookie"},
		{name: "--fields", args: []string{"GET", "/spaces/team/pages", "--fields", "items"}, wantMessage: "--fields"},
		{name: "GETに本文", args: []string{"GET", "/spaces/team/pages", "--input", "-"}, wantMessage: "--input"},
		{name: "読めない本文のファイル", args: []string{"POST", "/spaces/team/pages", "--input", "no-such-file.json"}, wantMessage: "no-such-file.json"},
		{name: "確認が要る操作に標準入力", args: []string{"DELETE", "/spaces/{space}/pages/12/draft_page", "--input", "-"}, wantMessage: "deleteDraftPage"},
		{name: "誤ったjq", args: []string{"GET", "/spaces/team/pages", "--jq", ".["}, wantMessage: "--jq"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := cmdtest.New(t)
			err := e.Run(NewCmdAPI, append([]string{"api"}, tt.args...)...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, tt.wantMessage) {
				t.Errorf("メッセージ = %q、含むべき文字列 = %q", cliErr.Message, tt.wantMessage)
			}
			assertNoRequest(t, e)
		})
	}
}

func TestAPI_ForbiddenHeaderHint(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	err := e.Run(NewCmdAPI, "api", "GET", "/spaces/team/pages", "-H", "Authorization: Bearer "+cmdtest.Token)
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
	if !strings.Contains(cliErr.Hint, "WIKINO_TOKEN") {
		t.Errorf("ヒントがトークンの渡し方を案内していない: %q", cliErr.Hint)
	}
	if strings.Contains(cliErr.Message+cliErr.Hint, cmdtest.Token) {
		t.Errorf("エラーにトークンが含まれる: %q / %q", cliErr.Message, cliErr.Hint)
	}
}

func TestAPI_DryRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantURL  string
		wantBody any
	}{
		{
			name:     "書き込み",
			args:     []string{"POST", "/spaces/{space}/pages", "--input", "-"},
			stdin:    `{"title":"議事録"}`,
			wantURL:  cmdtest.Host + pagesPath,
			wantBody: map[string]any{"title": "議事録"},
		},
		{
			name:    "読み取りも送らない",
			args:    []string{"GET", "/spaces/team/pages?limit=5"},
			wantURL: cmdtest.Host + pagesPath + "?limit=5",
		},
		{
			name:    "確認が要る操作も確認せずに出す",
			args:    []string{"DELETE", "/spaces/team/pages/12/draft_page", "-H", "If-Match: " + etag},
			wantURL: cmdtest.Host + draftPagePath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := cmdtest.New(t)
			e.Stdin = tt.stdin

			args := append(append([]string{"api"}, tt.args...), "--dry-run")
			if err := e.Run(NewCmdAPI, args...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			assertNoRequest(t, e)

			var got struct {
				DryRun   bool `json:"dry_run"`
				Requests []struct {
					Method  string            `json:"method"`
					URL     string            `json:"url"`
					Headers map[string]string `json:"headers"`
					Body    any               `json:"body"`
				} `json:"requests"`
			}
			cmdtest.Decode(t, e.Stdout.String(), &got)
			if !got.DryRun || len(got.Requests) != 1 {
				t.Fatalf("dry-runの出力 = %s", e.Stdout.String())
			}
			r := got.Requests[0]
			if r.URL != tt.wantURL || r.Method != strings.ToUpper(tt.args[0]) {
				t.Errorf("要求 = %s %s、期待値 = %s %s", r.Method, r.URL, tt.args[0], tt.wantURL)
			}
			if r.Headers["Authorization"] != "Bearer [REDACTED]" {
				t.Errorf("Authorizationが伏せられていない: %q", r.Headers["Authorization"])
			}
			if tt.wantBody != nil {
				if body, ok := r.Body.(map[string]any); !ok || body["title"] != "議事録" {
					t.Errorf("本文 = %v、期待値 = %v", r.Body, tt.wantBody)
				}
			}
			if strings.Contains(e.Stdout.String()+e.Stderr.String(), cmdtest.Token) {
				t.Errorf("出力にトークンが含まれる: %s", e.Stdout.String())
			}
			if !strings.Contains(e.Stderr.String(), "--dry-run") {
				t.Errorf("要求を送っていないことを標準エラーに出していない: %q", e.Stderr.String())
			}
		})
	}
}

// confirmationOutputは、確認プロトコルのJSON。
type confirmationOutput struct {
	Status         string   `json:"status"`
	Changes        []string `json:"changes"`
	ConfirmCommand string   `json:"confirm_command"`
}

func assertConfirmationRequired(t *testing.T, e *cmdtest.Env, err error) confirmationOutput {
	t.Helper()

	var silent *output.SilentError
	if !errors.As(err, &silent) || silent.ExitCode != output.ExitConfirmationRequired {
		t.Fatalf("終了コード9を期待したが %v だった", err)
	}
	assertNoRequest(t, e)

	var got confirmationOutput
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if got.Status != "confirmation_required" {
		t.Errorf("status = %q、期待値 = confirmation_required", got.Status)
	}
	if strings.Contains(e.Stdout.String(), cmdtest.Token) {
		t.Errorf("確認の出力にトークンが含まれる: %s", e.Stdout.String())
	}

	return got
}

func TestAPI_ConfirmationRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantChanges []string
		wantCommand string
	}{
		{
			name: "取り消せない操作",
			args: []string{"delete", "/spaces/{space}/pages/12/draft_page", "--header", "If-Match: " + etag},
			wantChanges: []string{
				"DELETE /spaces/team/pages/12/draft_page を送ります (deleteDraftPage)",
				"この操作は取り消せません",
			},
			wantCommand: `wikino api DELETE /spaces/team/pages/12/draft_page --header 'If-Match: "d41d8cd9"' --yes`,
		},
		{
			name:        "公開",
			args:        []string{"POST", "/spaces/team/pages/12/draft_page/publication", "--space", "team", "-H", "If-Match: " + etag},
			wantChanges: []string{"POST /spaces/team/pages/12/draft_page/publication を送ります (createDraftPagePublication)"},
			wantCommand: `wikino api POST /spaces/team/pages/12/draft_page/publication --space team --header 'If-Match: "d41d8cd9"' --yes`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := cmdtest.New(t)
			err := e.Run(NewCmdAPI, append([]string{"api"}, tt.args...)...)

			got := assertConfirmationRequired(t, e, err)
			if strings.Join(got.Changes, "\n") != strings.Join(tt.wantChanges, "\n") {
				t.Errorf("changes = %q、期待値 = %q", got.Changes, tt.wantChanges)
			}
			if got.ConfirmCommand != tt.wantCommand {
				t.Errorf("confirm_command = %q、期待値 = %q", got.ConfirmCommand, tt.wantCommand)
			}
		})
	}
}

func TestAPI_ConfirmationWithInputFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	e := cmdtest.New(t)
	err := e.Run(NewCmdAPI, "api", "POST", "/spaces/team/pages/12/draft_page/publication", "--input", path)

	got := assertConfirmationRequired(t, e, err)
	want := "wikino api POST /spaces/team/pages/12/draft_page/publication --input " + path + " --yes"
	if got.ConfirmCommand != want {
		t.Errorf("confirm_command = %q、期待値 = %q", got.ConfirmCommand, want)
	}
}

func TestAPI_ConfirmationPrompt(t *testing.T) {
	t.Parallel()

	// TTYで対話できるときは、標準エラーで尋ね、承認されなければ送らない。
	e := cmdtest.New(t)
	e.Stdin = "n\n"
	err := e.RunTTY(NewCmdAPI, "api", "DELETE", "/spaces/team/pages/12/draft_page", "-H", "If-Match: "+etag)
	_ = cmdtest.AssertCLIError(t, err, output.CodeConfirmationDeclined, output.ExitConfirmationRequired)
	assertNoRequest(t, e)
	if !strings.Contains(e.Stderr.String(), "DELETE /spaces/team/pages/12/draft_page を送ります") {
		t.Errorf("標準エラーに確認の内容が無い: %q", e.Stderr.String())
	}
}

func TestAPI_ConfirmationNotRequired(t *testing.T) {
	t.Parallel()

	// 確認が要らない書き込みは、`--yes` が無くても送る。
	e := cmdtest.New(t)
	handleRecord(e, http.MethodPatch, draftPagePath, func() *http.Response {
		return rawResponse(http.StatusOK, "application/json", `{}`)
	})
	e.Stdin = `{"title": "a"}`

	if err := e.Run(NewCmdAPI, "api", "PATCH", "/spaces/team/pages/12/draft_page", "--input", "-"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if got := e.Stdout.String(); got != "{}" {
		t.Errorf("標準出力 = %q、期待値 = %q", got, "{}")
	}
}

func TestAPI_InvalidHeadersDoNotExposeSecrets(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "X-Test"} {
		for _, shape := range []struct {
			name  string
			value string
		}{
			{name: "途中の改行", value: name + ": " + cmdtest.Token + "\r\nX-Other: value"},
			{name: "名前の前後の空白", value: " " + name + " : " + cmdtest.Token},
			{name: "コロンの欠落", value: name + " " + cmdtest.Token},
			{name: "名前に秘密が混入", value: name + " " + cmdtest.Token + ": value"},
		} {
			for _, lang := range []string{"ja", "en"} {
				t.Run(name+"/"+shape.name+"/"+lang, func(t *testing.T) {
					t.Parallel()

					e := cmdtest.New(t)
					e.Vars["WIKINO_LANG"] = lang
					err := e.Run(NewCmdAPI, "api", "GET", "/spaces/team/pages", "-H", shape.value)
					cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
					assertNoRequest(t, e)
					if strings.Contains(cliErr.Message+cliErr.Hint+err.Error(), cmdtest.Token) {
						t.Error("エラーまたはヒントに秘密の値が含まれている")
					}

					for _, jsonOutput := range []bool{false, true} {
						var stderr bytes.Buffer
						if err := output.WriteError(&stderr, err, jsonOutput); err != nil {
							t.Fatalf("エラーの表示に失敗した: %v", err)
						}
						if strings.Contains(stderr.String()+e.Stdout.String()+e.Stderr.String(), cmdtest.Token) {
							t.Errorf("最終出力に秘密の値が含まれている (JSON: %t)", jsonOutput)
						}
					}
				})
			}
		}
	}
}
