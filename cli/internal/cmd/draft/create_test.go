package draft

import (
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// handleCreateは、下書きの作成の要求を受けて送った本文をsentに記録し、作成した下書きを返す。
// 新しいページの下書きでは、ページ番号を42にする。
func handleCreate(t *testing.T, e *cmdtest.Env, sent *map[string]any) {
	t.Helper()

	e.Handle(http.MethodPost, draftPagesPath, func(req *http.Request) *http.Response {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("要求の本文を読めない: %v", err)
		}
		*sent = map[string]any{}
		cmdtest.Decode(t, string(data), sent)

		number := int32(42)
		if n, ok := (*sent)["page_number"].(float64); ok {
			number = int32(n)
		}
		var title *string
		if s, ok := (*sent)["title"].(string); ok {
			title = &s
		}
		body, _ := (*sent)["body"].(string)
		return cmdtest.JSON(http.StatusCreated, testDraft(number, title, body), "ETag", etag, "Location", draftPath(number))
	})
}

// handlePageは、既存のページnumberの取得の要求に、トピックtopicのページで応答する。
func handlePage(e *cmdtest.Env, number, topic int32) {
	e.Handle(http.MethodGet, pagesPath+"/"+formatNumber(number), func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testPage(number, topic), "ETag", `"page-etag"`)
	})
}

func TestCreate_Request(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		stdin     string
		wantSent  map[string]any
		wantPaths []string
		// wantPreviewTopicは、プレビューで送ったトピック番号。0ならプレビューを送らない。
		wantPreviewTopic float64
	}{
		{
			// 本文とタイトルを省くと、APIがページの今の内容を使う。本文が無いためプレビューもしない。
			name:      "既存のページの今の内容",
			args:      []string{"12"},
			wantSent:  map[string]any{"page_number": float64(12)},
			wantPaths: []string{"POST " + draftPagesPath},
		},
		{
			// プレビューのトピックは、ページを取得して得る。
			name:             "既存のページに本文を渡す",
			args:             []string{"12", "--title", "議事録 (改訂)", "--body", "# 議事録\n\n[[予算案]]\n"},
			wantSent:         map[string]any{"page_number": float64(12), "title": "議事録 (改訂)", "body": "# 議事録\n\n[[予算案]]\n"},
			wantPaths:        []string{"GET " + pagesPath + "/12", "POST " + previewPath, "POST " + draftPagesPath},
			wantPreviewTopic: 5,
		},
		{
			name:             "既存のページに標準入力の本文を渡す",
			args:             []string{"12", "--body-file", "-"},
			stdin:            "本文\n\n",
			wantSent:         map[string]any{"page_number": float64(12), "body": "本文\n\n"},
			wantPaths:        []string{"GET " + pagesPath + "/12", "POST " + previewPath, "POST " + draftPagesPath},
			wantPreviewTopic: 5,
		},
		{
			name:             "新しいページ",
			args:             []string{"--topic", "3", "--title", "議事録", "--body", "# 議事録\n"},
			wantSent:         map[string]any{"topic_number": float64(3), "title": "議事録", "body": "# 議事録\n"},
			wantPaths:        []string{"POST " + previewPath, "POST " + draftPagesPath},
			wantPreviewTopic: 3,
		},
		{
			// タイトルと本文を省くと、タイトルの無い空の下書きになる。
			name:      "新しいページのタイトルと本文を省く",
			args:      []string{"--topic", "3"},
			wantSent:  map[string]any{"topic_number": float64(3)},
			wantPaths: []string{"POST " + draftPagesPath},
		},
		{
			// 空の本文はリンクを持たないため、プレビューを送らない。
			name:      "新しいページの空の本文",
			args:      []string{"--topic", "3", "--body", ""},
			wantSent:  map[string]any{"topic_number": float64(3), "body": ""},
			wantPaths: []string{"POST " + draftPagesPath},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Stdin = tt.stdin
			handlePage(e, 12, 5)
			preview := handlePreview(t, e)
			var sent map[string]any
			handleCreate(t, e, &sent)

			if err := e.Run(NewCmdDraft, append([]string{"draft", "create"}, tt.args...)...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			if len(sent) != len(tt.wantSent) {
				t.Errorf("送った本文 = %v、期待値 = %v", sent, tt.wantSent)
			}
			for key, value := range tt.wantSent {
				if sent[key] != value {
					t.Errorf("送った %s = %v、期待値 = %v", key, sent[key], value)
				}
			}
			if got := requestPaths(e); !slices.Equal(got, tt.wantPaths) {
				t.Errorf("送った要求 = %v、期待値 = %v", got, tt.wantPaths)
			}
			if tt.wantPreviewTopic != 0 {
				previews := preview.sent()
				if len(previews) != 1 || previews[0]["topic_number"] != tt.wantPreviewTopic || previews[0]["body"] != tt.wantSent["body"] {
					t.Errorf("プレビューで送った本文 = %v", previews)
				}
			}
		})
	}
}

func TestCreate_JSON(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePreview(t, e)
	var sent map[string]any
	handleCreate(t, e, &sent)

	if err := e.Run(NewCmdDraft, "draft", "create", "--topic", "3", "--title", "議事録", "--body", "# 議事録\n"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	var got map[string]any
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if got["etag"] != etag || got["page_number"] != float64(42) || got["body"] != "# 議事録\n" || got["title"] != "議事録" {
		t.Errorf("出力 = %v", got)
	}
	if e.Stderr.Len() > 0 {
		t.Errorf("JSONの出力で標準エラーに出力がある: %q", e.Stderr.String())
	}
}

func TestCreate_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePage(e, 12, 3)
	handlePreview(t, e)
	var sent map[string]any
	handleCreate(t, e, &sent)

	if err := e.RunTTY(NewCmdDraft, "draft", "create", "12", "--title", "議事録", "--body", "# 議事録\n\n本文\n"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "create", e.Stdout.String())
	if got, want := e.Stderr.String(), "ページ12の下書きを作成しました。\n"; got != want {
		t.Errorf("標準エラー = %q、期待値 = %q", got, want)
	}
}

func TestCreate_CreatedWikilinks(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePreview(t, e, newPageLink("予算案"))
	var sent map[string]any
	handleCreate(t, e, &sent)

	if err := e.Run(NewCmdDraft, "draft", "create", "--topic", "3", "--body", "[[予算案]]\n"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// 作られた見込みのリンク先は、JSONの出力でも標準エラーに出し、標準出力を汚さない。
	if !strings.Contains(e.Stderr.String(), "開発/予算案") {
		t.Errorf("標準エラーに作られたリンク先が無い: %q", e.Stderr.String())
	}
	var got map[string]any
	cmdtest.Decode(t, e.Stdout.String(), &got)
}

func TestCreate_DryRun(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePage(e, 12, 3)
	handlePreview(t, e, newPageLink("予算案"))
	if err := e.Run(NewCmdDraft, "draft", "create", "12", "--body", "[[予算案]]\n", "--dry-run"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	// ページの取得とWikiリンクのプレビュー (読み取り) だけを送り、作成の要求は送らない。
	if got, want := requestPaths(e), []string{"GET " + pagesPath + "/12", "POST " + previewPath}; !slices.Equal(got, want) {
		t.Errorf("送った要求 = %v、期待値 = %v", got, want)
	}

	var got struct {
		DryRun   bool `json:"dry_run"`
		Requests []struct {
			Method string         `json:"method"`
			URL    string         `json:"url"`
			Body   map[string]any `json:"body"`
		} `json:"requests"`
		Wikilinks []map[string]any `json:"wikilinks"`
	}
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if !got.DryRun || len(got.Requests) != 1 {
		t.Fatalf("出力 = %s", e.Stdout.String())
	}
	req := got.Requests[0]
	if req.Method != http.MethodPost || req.URL != cmdtest.Host+draftPagesPath {
		t.Errorf("要求 = %s %s", req.Method, req.URL)
	}
	if req.Body["page_number"] != float64(12) || req.Body["body"] != "[[予算案]]\n" {
		t.Errorf("要求の本文 = %v", req.Body)
	}
	if len(got.Wikilinks) != 1 || got.Wikilinks[0]["resolution"] != "new_page" {
		t.Errorf("wikilinks = %v", got.Wikilinks)
	}
	assertNoSecret(t, e)
	if !strings.Contains(e.Stderr.String(), "--dry-run") {
		t.Errorf("標準エラーに要求を送っていないことの案内が無い: %q", e.Stderr.String())
	}
}

func TestCreate_UsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{name: "ページ番号と--topicの併用", args: []string{"12", "--topic", "3"}, wantMessage: "--topic"},
		{name: "ページ番号も--topicも無い", args: []string{"--title", "議事録"}, wantMessage: "--topic"},
		{name: "不正なページ番号", args: []string{"0"}, wantMessage: "<page_number>"},
		{name: "不正なトピック番号", args: []string{"--topic", "abc"}, wantMessage: "--topic"},
		{name: "位置引数が多い", args: []string{"12", "13"}},
		{name: "本文のフラグの併用", args: []string{"12", "--body", "a", "--body-file", "-"}, wantMessage: "--body, --body-file"},
		{name: "--dry-runと--fieldsの併用", args: []string{"12", "--dry-run", "--fields", "page_number"}, wantMessage: "--dry-run, --fields"},
		{name: "未知の項目", args: []string{"12", "--fields", "number"}, wantMessage: "number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdDraft, append([]string{"draft", "create"}, tt.args...)...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, tt.wantMessage) {
				t.Errorf("メッセージ %q に %q が無い", cliErr.Message, tt.wantMessage)
			}
			assertNoRequest(t, e)
		})
	}
}

func TestCreate_APIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		status       int
		wantCode     string
		wantExitCode int
		wantHint     string
		pointer      string
	}{
		{name: "下書きが既にある", args: []string{"12"}, status: http.StatusConflict, wantCode: output.CodeConflict, wantExitCode: output.ExitConflict, wantHint: "`wikino draft update 12`"},
		{name: "既存のページが無い", pointer: "#/page_number", args: []string{"12"}, status: http.StatusUnprocessableEntity, wantCode: output.CodeValidationFailed, wantExitCode: output.ExitValidation, wantHint: "`wikino page list`"},
		{name: "既存のページに作成できない", args: []string{"12"}, status: http.StatusNotFound, wantCode: output.CodeNotFound, wantExitCode: output.ExitNotFound, wantHint: "ページ12"},
		{name: "既存のページの5xx", args: []string{"12"}, status: http.StatusInternalServerError, wantCode: output.CodeServerError, wantExitCode: output.ExitError, wantHint: "`wikino draft get 12`"},
		{name: "トピックが無い", pointer: "#/topic_number", args: []string{"--topic", "3"}, status: http.StatusUnprocessableEntity, wantCode: output.CodeValidationFailed, wantExitCode: output.ExitValidation, wantHint: "`wikino topic list`"},
		{name: "トピックに作成できない", args: []string{"--topic", "3"}, status: http.StatusNotFound, wantCode: output.CodeNotFound, wantExitCode: output.ExitNotFound, wantHint: "トピック3"},
		{name: "新しいページの5xxは作成済みかを確かめさせる", args: []string{"--topic", "3"}, status: http.StatusInternalServerError, wantCode: output.CodeServerError, wantExitCode: output.ExitError, wantHint: "`wikino draft list`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodPost, draftPagesPath, func(req *http.Request) *http.Response {
				if tt.pointer != "" {
					return cmdtest.JSON(tt.status, map[string]any{
						"detail": "失敗しました。",
						"errors": []map[string]any{{"pointer": tt.pointer, "detail": "対象が見つかりません。"}},
					})
				}
				return cmdtest.Problem(tt.status, "失敗しました。")
			})

			err := e.Run(NewCmdDraft, append([]string{"draft", "create"}, tt.args...)...)
			cliErr := cmdtest.AssertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒント %q に %q が無い", cliErr.Hint, tt.wantHint)
			}
		})
	}
}

func TestCreate_PageNotFoundBeforePreview(t *testing.T) {
	t.Parallel()

	// 本文を渡した既存のページが無ければ、プレビューも作成の要求も送らずに失敗する。
	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath+"/99", func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "ページが見つかりません。")
	})

	err := e.Run(NewCmdDraft, "draft", "create", "99", "--body", "本文")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "wikino page list") {
		t.Errorf("ヒントにページの一覧のコマンドが無い: %q", cliErr.Hint)
	}
	if got, want := requestPaths(e), []string{"GET " + pagesPath + "/99"}; !slices.Equal(got, want) {
		t.Errorf("送った要求 = %v、期待値 = %v", got, want)
	}
}

func TestCreate_ValidationHints(t *testing.T) {
	t.Parallel()

	for _, target := range []struct {
		name        string
		args        []string
		field       string
		listCommand string
	}{
		{name: "既存のページ", args: []string{"12", "--title", ""}, field: "page_number", listCommand: "wikino page list"},
		{name: "新しいページ", args: []string{"--topic", "3", "--title", ""}, field: "topic_number", listCommand: "wikino topic list"},
	} {
		t.Run(target.name, func(t *testing.T) {
			t.Parallel()

			for _, tt := range []struct {
				name           string
				pointer        any
				wantNumberHint bool
			}{
				{name: "空のタイトル", pointer: "#/title"},
				{name: "本文", pointer: "#/body"},
				{name: "項目を特定できない", pointer: nil},
				{name: "作成先の番号", pointer: "#/" + target.field, wantNumberHint: true},
				{name: "フラグメントでない番号のポインター", pointer: "/" + target.field, wantNumberHint: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()

					e := newEnv(t)
					e.Handle(http.MethodPost, draftPagesPath, func(req *http.Request) *http.Response {
						return cmdtest.JSON(http.StatusUnprocessableEntity, map[string]any{
							"detail": "入力を確認してください。",
							"errors": []map[string]any{{"pointer": tt.pointer, "detail": "入力が不正です。"}},
						})
					})

					err := e.Run(NewCmdDraft, append([]string{"draft", "create"}, target.args...)...)
					cliErr := cmdtest.AssertCLIError(t, err, output.CodeValidationFailed, output.ExitValidation)
					wantHint := i18n.NewLocalizer("ja").T("error_api_validation_failed_hint")
					if tt.wantNumberHint {
						if !strings.Contains(cliErr.Hint, target.listCommand) {
							t.Errorf("ヒントに番号を確認するコマンドが無い: %q", cliErr.Hint)
						}
					} else if cliErr.Hint != wantHint {
						t.Errorf("ヒント = %q、期待値 = %q", cliErr.Hint, wantHint)
					}
					if !strings.Contains(cliErr.Message, "入力が不正です。") {
						t.Errorf("エラーのメッセージに項目の説明が無い: %q", cliErr.Message)
					}
				})
			}
		})
	}
}
