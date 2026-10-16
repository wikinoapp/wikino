package draft

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func handleDraft(e *cmdtest.Env, number int32, draft map[string]any) {
	e.Handle(http.MethodGet, draftPath(number), func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, draft, "ETag", etag)
	})
}

func TestGet_JSON(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handleDraft(e, 12, testDraft(12, ptr("議事録"), "# 議事録\n\n本文\n"))

	if err := e.Run(NewCmdDraft, "draft", "get", "12"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	var got map[string]any
	cmdtest.Decode(t, e.Stdout.String(), &got)
	want := map[string]any{
		"etag":         etag,
		"page_number":  float64(12),
		"topic_number": float64(3),
		"topic_name":   "トピック3",
		"url":          "https://wikino.example.com/s/team/pages/12/edit",
		"body":         "# 議事録\n\n本文\n",
		"title":        "議事録",
		"modified_at":  "2026-10-01T09:00:00.123456Z",
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
	handleDraft(e, 12, testDraft(12, ptr("議事録"), "# 議事録\n\n- [ ] 予算案\n"))

	if err := e.RunTTY(NewCmdDraft, "draft", "get", "12"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "get", e.Stdout.String())
}

func TestGet_Markdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		draft map[string]any
		want  string
	}{
		{
			name:  "タイトルのある下書き",
			draft: testDraft(12, ptr("議事録: 10月"), "# 議事録\n"),
			want: "---\n" +
				`wikino_title: "議事録: 10月"` + "\n" +
				"wikino_number: 12\n" +
				`wikino_etag: "\"d41d8cd9\""` + "\n" +
				"wikino_topic_number: 3\n" +
				`wikino_modified_at: "2026-10-01T09:00:00.123456Z"` + "\n" +
				`wikino_url: "https://wikino.example.com/s/team/pages/12/edit"` + "\n" +
				"---\n\n# 議事録\n",
		},
		{
			// タイトルの無い下書きは `wikino_title` を書かず、書き戻しでタイトルを変えない。
			name:  "タイトルの無い下書き",
			draft: testDraft(30, nil, ""),
			want: "---\n" +
				"wikino_number: 30\n" +
				`wikino_etag: "\"d41d8cd9\""` + "\n" +
				"wikino_topic_number: 3\n" +
				`wikino_modified_at: "2026-10-01T09:00:00.123456Z"` + "\n" +
				`wikino_url: "https://wikino.example.com/s/team/pages/30/edit"` + "\n" +
				"---\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			number := tt.draft["page_number"].(int32)
			handleDraft(e, number, tt.draft)

			if err := e.Run(NewCmdDraft, "draft", "get", formatNumber(number), "--md"); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			got := e.Stdout.String()
			if got != tt.want {
				t.Errorf("標準出力 = %q、期待値 = %q", got, tt.want)
			}
			// 下書きには公開状態が無いため、`wikino_status` を書かない。
			if strings.Contains(got, "wikino_status") {
				t.Errorf("下書きのページファイルに公開状態がある: %q", got)
			}
		})
	}
}

func TestGet_MarkdownWithOutputFlags(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"--json"},
		{"--fields", "page_number"},
		{"--jq", ".body"},
	} {
		t.Run(args[0], func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdDraft, append([]string{"draft", "get", "12", "--md"}, args...)...)
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
	e.Handle(http.MethodGet, draftPath(99), func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "下書きが見つかりません。")
	})

	err := e.Run(NewCmdDraft, "draft", "get", "99")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "wikino draft list") {
		t.Errorf("ヒントに下書きの一覧のコマンドが無い: %q", cliErr.Hint)
	}
}

func TestGet_InvalidNumber(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"0", "abc", "../12"} {
		t.Run(arg, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdDraft, "draft", "get", arg)
			_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			assertNoRequest(t, e)
		})
	}
}
