package draft

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const revisionID = "00000000-0000-0000-0000-000000000001"

func handleRevision(e *cmdtest.Env, revision map[string]any) {
	e.Handle(http.MethodGet, revisionsPath+"/"+revisionID, func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, revision)
	})
}

func TestRevisionGet_JSON(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handleRevision(e, testRevision(1, ptr("議事録"), "# 議事録\n\n本文\n"))

	if err := e.Run(NewCmdDraft, "draft", "revision", "get", "12", revisionID); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	var got map[string]any
	cmdtest.Decode(t, e.Stdout.String(), &got)
	want := map[string]any{
		"id":         revisionID,
		"title":      "議事録",
		"body":       "# 議事録\n\n本文\n",
		"created_at": "2026-10-01T09:00:00.123456Z",
	}
	if len(got) != len(want) {
		t.Errorf("出力 = %v、期待値 = %v", got, want)
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %v、期待値 = %v", key, got[key], value)
		}
	}
}

func TestRevisionGet_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handleRevision(e, testRevision(1, ptr("議事録"), "# 議事録\n\n- [ ] 予算案\n"))

	if err := e.RunTTY(NewCmdDraft, "draft", "revision", "get", "12", revisionID); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "revision_get", e.Stdout.String())
}

func TestRevisionGet_Markdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		title *string
		body  string
		want  string
	}{
		{
			name:  "タイトルのあるリビジョンはwikino_titleだけを書く",
			title: ptr("議事録: 10月"),
			body:  "# 議事録\n",
			want:  "---\nwikino_title: \"議事録: 10月\"\n---\n\n# 議事録\n",
		},
		{
			name: "タイトルの無いリビジョンは本文だけを出す",
			body: "# 下書き\n",
			want: "# 下書き\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			handleRevision(e, testRevision(1, tt.title, tt.body))

			if err := e.RunTTY(NewCmdDraft, "draft", "revision", "get", "12", revisionID, "--md"); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			// リビジョンは版を持たないため、`wikino_etag` を書かず、`--md-file` で書き戻せない。
			if got := e.Stdout.String(); got != tt.want {
				t.Errorf("出力 = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestRevisionGet_NotFound(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, revisionsPath+"/"+revisionID, func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "リビジョンが見つかりません。")
	})

	err := e.Run(NewCmdDraft, "draft", "revision", "get", "12", revisionID)
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "`wikino draft revision list 12`") {
		t.Errorf("ヒントに編集履歴の一覧のコマンドが無い: %q", cliErr.Hint)
	}
}

func TestRevisionGet_UsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{name: "リビジョンのIDが無い", args: []string{"12"}, wantMessage: "2"},
		{name: "不正なページ番号", args: []string{"x", revisionID}, wantMessage: "<page_number>"},
		{name: "UUIDでないリビジョンのID", args: []string{"12", "../1"}, wantMessage: "<revision_id>"},
		{name: "--mdと--jqの併用", args: []string{"12", revisionID, "--md", "--jq", ".body"}, wantMessage: "--md"},
		{name: "未知の項目", args: []string{"12", revisionID, "--fields", "etag"}, wantMessage: "etag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdDraft, append([]string{"draft", "revision", "get"}, tt.args...)...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, tt.wantMessage) {
				t.Errorf("メッセージ %q に %q が無い", cliErr.Message, tt.wantMessage)
			}
			assertNoRequest(t, e)
		})
	}
}
