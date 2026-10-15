package page

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// writeFileは、一時ディレクトリにcontentのファイルを作ってパスを返す。
func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

// exportPageは、`page get 12 --md` でbodyのページを書き出したページファイルを返す。
func exportPage(t *testing.T, body string) string {
	t.Helper()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagePath, func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testPage(12, ptr("議事録"), body), "ETag", etag)
	})
	if err := e.Run(NewCmdPage, "page", "get", "12", "--md"); err != nil {
		t.Fatalf("書き出しでエラーを期待しなかったが %v だった", err)
	}

	return e.Stdout.String()
}

// `--md` で書き出したファイルを編集せずに書き戻すと、本文・タイトル・版がそのまま送られる。
func TestUpdate_MarkdownFileRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "通常の本文", body: "# 議事録\n\n- [ ] 予算案\n"},
		{name: "先頭が水平線の本文", body: "---\n\n本文\n\n---\n"},
		{name: "本文自身のfrontmatter", body: "---\ntags: [議事録]   # 分類\naliases: []\n---\n\n本文\n"},
		{name: "フロー形式の本文自身のfrontmatter", body: "---\n{tags: [議事録]}\n---\n\n本文\n"},
		{name: "末尾に改行の無い本文", body: "本文"},
		{name: "CRLFの本文", body: "本文\r\n続き\r\n"},
		{name: "空の本文", body: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writeFile(t, "page.md", exportPage(t, tt.body))

			e := newEnv(t)
			var sent sentRequest
			handleUpdate(t, e, &sent)
			if err := e.Run(NewCmdPage, "page", "update", "12", "--md-file", path); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			if got := sent.body["body"]; got != tt.body {
				t.Errorf("送った本文 = %q、期待値 = %q", got, tt.body)
			}
			if got := sent.body["title"]; got != "議事録" {
				t.Errorf("送ったタイトル = %v、期待値 = 議事録", got)
			}
			if sent.ifMatch != etag {
				t.Errorf("If-Match = %q、期待値 = %q (ファイルのwikino_etag)", sent.ifMatch, etag)
			}
		})
	}
}

func TestUpdate_MarkdownFile(t *testing.T) {
	t.Parallel()

	t.Run("編集したタイトルと本文を送り、読み取り専用のキーは無視する", func(t *testing.T) {
		t.Parallel()

		content := "---\n" +
			"wikino_title: \"定例会議の議事録\"\n" +
			"wikino_number: 12\n" +
			"wikino_etag: \"\\\"d41d8cd9\\\"\"\n" +
			"wikino_topic_number: 99\n" +
			"wikino_status: \"unpublished\"\n" +
			"---\n\n# 議事録\n\n- [x] 予算案\n"
		path := writeFile(t, "page.md", content)

		e := newEnv(t)
		var sent sentRequest
		handleUpdate(t, e, &sent)
		preview := handlePreview(t, e)
		if err := e.Run(NewCmdPage, "page", "update", "12", "--md-file", path); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}

		want := map[string]any{"title": "定例会議の議事録", "body": "# 議事録\n\n- [x] 予算案\n"}
		if len(sent.body) != len(want) || sent.body["title"] != want["title"] || sent.body["body"] != want["body"] {
			t.Errorf("送った内容 = %v、期待値 = %v", sent.body, want)
		}
		// トピックはファイルの値ではなく、取得したページの値で確かめる。
		if got := preview.requests; len(got) != 1 || got[0]["topic_number"] != float64(3) {
			t.Errorf("プレビューの要求 = %v、期待値 = トピック3", got)
		}
	})

	t.Run("wikino_titleが無ければタイトルを変えない", func(t *testing.T) {
		t.Parallel()

		path := writeFile(t, "page.md", "---\nwikino_number: 12\nwikino_etag: \"\\\"d41d8cd9\\\"\"\n---\n\n本文\n")

		e := newEnv(t)
		var sent sentRequest
		handleUpdate(t, e, &sent)
		if err := e.Run(NewCmdPage, "page", "update", "12", "--md-file", path); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		if _, ok := sent.body["title"]; ok || sent.body["body"] != "本文\n" {
			t.Errorf("送った内容 = %v、期待値 = 本文だけ", sent.body)
		}
	})

	t.Run("標準入力から読む", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		e.Stdin = exportPage(t, "本文\n")
		var sent sentRequest
		handleUpdate(t, e, &sent)
		if err := e.Run(NewCmdPage, "page", "update", "12", "--md-file", "-"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		if sent.body["body"] != "本文\n" || sent.ifMatch != etag {
			t.Errorf("送った内容 = %v、If-Match = %q", sent.body, sent.ifMatch)
		}
	})

	t.Run("dry-runでファイルの版を送る要求を出す", func(t *testing.T) {
		t.Parallel()

		path := writeFile(t, "page.md", exportPage(t, "本文\n"))

		e := newEnv(t)
		handleCurrentPage(e, `"latest"`)
		handlePreview(t, e)
		if err := e.Run(NewCmdPage, "page", "update", "12", "--md-file", path, "--dry-run"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}

		got := decodeDryRun(t, e.Stdout.String())
		if len(got.Requests) != 1 || got.Requests[0].Method != http.MethodPatch || got.Requests[0].Headers["If-Match"] != etag {
			t.Errorf("送る要求 = %+v、期待値 = ファイルの版のPATCH", got.Requests)
		}
		if got.Requests[0].Body["body"] != "本文\n" {
			t.Errorf("送る本文 = %v", got.Requests[0].Body)
		}
	})
}

func TestUpdate_MarkdownFileUsageErrors(t *testing.T) {
	t.Parallel()

	exported := exportPage(t, "本文\n")
	valid := writeFile(t, "page.md", exported)
	otherPage := writeFile(t, "other.md", strings.Replace(exported, "wikino_number: 12", "wikino_number: 13", 1))
	noETag := writeFile(t, "no-etag.md", "---\nwikino_number: 12\nwikino_title: \"議事録\"\n---\n\n本文\n")
	noFrontmatter := writeFile(t, "plain.md", "# 議事録\n")
	wildcard := writeFile(t, "wildcard.md", "---\nwikino_number: 12\nwikino_etag: \"*\"\n---\n\n本文\n")
	stringNumber := writeFile(t, "string-number.md", "---\nwikino_number: \"12\"\nwikino_etag: \"v1\"\n---\n\n本文\n")
	invalidUTF8 := writeFile(t, "invalid.md", "---\nwikino_etag: \"v1\"\n---\n\xff\n")

	tests := []struct {
		name        string
		args        []string
		wantMessage string
		wantHint    string
	}{
		{name: "別のページのファイル", args: []string{"--md-file", otherPage}, wantMessage: "(13)", wantHint: "`wikino page get 12 --md > page.md`"},
		{name: "版の無いファイル", args: []string{"--md-file", noETag}, wantMessage: "`wikino_etag`", wantHint: "`wikino page get 12 --md > page.md`"},
		{name: "frontmatterの無いファイル", args: []string{"--md-file", noFrontmatter}, wantMessage: "`wikino_etag`"},
		{name: "版を問わない版", args: []string{"--md-file", wildcard}, wantMessage: "`wikino_etag` に `*`"},
		{name: "番号が文字列", args: []string{"--md-file", stringNumber}, wantMessage: "`wikino_number`"},
		{name: "UTF-8でないファイル", args: []string{"--md-file", invalidUTF8}, wantMessage: "UTF-8"},
		{name: "無いファイル", args: []string{"--md-file", filepath.Join(t.TempDir(), "missing.md")}, wantMessage: "missing.md"},
		{name: "--bodyとの併用", args: []string{"--md-file", valid, "--body", "本文"}, wantMessage: "--md-file, --body"},
		{name: "--body-fileとの併用", args: []string{"--md-file", valid, "--body-file", valid}, wantMessage: "--md-file, --body-file"},
		{name: "--titleとの併用", args: []string{"--md-file", valid, "--title", "議事録"}, wantMessage: "--md-file, --title"},
		{name: "--if-matchとの併用", args: []string{"--md-file", valid, "--if-match", etag}, wantMessage: "--md-file, --if-match"},
		{name: "--forceとの併用", args: []string{"--md-file", valid, "--force"}, wantMessage: "--md-file, --force"},
		{name: "--editsとの併用", args: []string{"--md-file", valid, "--edits", valid}, wantMessage: "--md-file, --edits"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdPage, append([]string{"page", "update", "12"}, tt.args...)...)
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

func TestUpdate_MarkdownFilePreconditionFailed(t *testing.T) {
	t.Parallel()

	path := writeFile(t, "page.md", exportPage(t, "本文\n"))

	e := newEnv(t)
	handleCurrentPage(e, `"latest"`)
	handlePreview(t, e)
	e.Handle(http.MethodPatch, pagePath, func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusPreconditionFailed, "ページはその後に更新されています。")
	})

	err := e.Run(NewCmdPage, "page", "update", "12", "--md-file", path)
	cliErr := cmdtest.AssertCLIError(t, err, output.CodePreconditionFailed, output.ExitConflict)
	// 版はファイルから読むため、`--if-match` ではなくファイルの書き出し直しを案内する。
	if !strings.Contains(cliErr.Hint, "`wikino page get 12 --md > page.md`") || strings.Contains(cliErr.Hint, "--if-match") {
		t.Errorf("ヒント %q にファイルを書き出し直すコマンドが無い", cliErr.Hint)
	}
}
