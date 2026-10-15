package page

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// wikilinkDryRunOutputは、本文を書き込むコマンドの `--dry-run` の標準出力。
type wikilinkDryRunOutput struct {
	DryRun   bool `json:"dry_run"`
	Requests []struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"requests"`
	Wikilinks []struct {
		Text       string         `json:"text"`
		TopicName  string         `json:"topic_name"`
		PageTitle  string         `json:"page_title"`
		Resolution string         `json:"resolution"`
		Page       map[string]any `json:"page"`
	} `json:"wikilinks"`
}

func TestCreate_WikilinkPreviewDryRun(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	preview := handlePreview(t, e,
		previewLink("議事録", "existing_page"),
		previewLink("予算案", "new_page"),
		previewLink("定例会議", "new_page"),
		previewLink("無いトピック/ページ", "topic_not_found"),
	)

	const body = "[[議事録]] [[予算案]] [[定例会議]] [[無いトピック/ページ]]\n"
	if err := e.Run(NewCmdPage, "page", "create", "--topic", "3", "--title", "メモ", "--body", body, "--dry-run"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// プレビューには、書き込むトピックと本文をそのまま送る。
	if got := preview.bodies(); len(got) != 1 || got[0] != body {
		t.Errorf("プレビューに送った本文 = %q、期待値 = %q", got, body)
	}
	if got := preview.requests[0]["topic_number"]; got != float64(3) {
		t.Errorf("プレビューに送ったトピック番号 = %v、期待値 = 3", got)
	}

	var got wikilinkDryRunOutput
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if !got.DryRun || len(got.Requests) != 1 || got.Requests[0].Method != http.MethodPost {
		t.Errorf("送る要求 = %+v", got)
	}
	wantResolutions := []string{"existing_page", "new_page", "new_page", "topic_not_found"}
	if len(got.Wikilinks) != len(wantResolutions) {
		t.Fatalf("wikilinks = %+v、期待値 = %d 件", got.Wikilinks, len(wantResolutions))
	}
	for i, want := range wantResolutions {
		if got.Wikilinks[i].Resolution != want {
			t.Errorf("wikilinks[%d].resolution = %q、期待値 = %q", i, got.Wikilinks[i].Resolution, want)
		}
	}
	if got.Wikilinks[0].Page["number"] != float64(20) {
		t.Errorf("既存のページ = %v", got.Wikilinks[0].Page)
	}

	wantStderr := "書き込むと、存在しないリンク先2件を未公開のページとして作成します: 開発/予算案, 開発/定例会議\n" +
		"`--dry-run` のため、要求を送っていません。\n"
	if e.Stderr.String() != wantStderr {
		t.Errorf("標準エラー = %q、期待値 = %q", e.Stderr.String(), wantStderr)
	}
}

func TestCreate_WikilinkPreviewDryRunWithoutLinks(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePreview(t, e)
	if err := e.Run(NewCmdPage, "page", "create", "--topic", "3", "--title", "メモ", "--body", "本文\n", "--dry-run"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// リンクが無くても `wikilinks` を空の配列で出し、項目の有無で分岐させない。
	if !strings.Contains(e.Stdout.String(), `"wikilinks":[]`) {
		t.Errorf("標準出力に空のwikilinksが無い: %s", e.Stdout.String())
	}
	if strings.Contains(e.Stderr.String(), "未公開のページ") {
		t.Errorf("作られるリンク先が無いのに案内がある: %q", e.Stderr.String())
	}
}

func TestCreate_EmptyBodySkipsPreview(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	var sent sentRequest
	handleCreate(t, e, &sent)

	if err := e.Run(NewCmdPage, "page", "create", "--topic", "3", "--title", "メモ", "--body", ""); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if requests := e.Requests(); len(requests) != 1 || requests[0].URL.Path != pagesPath {
		t.Errorf("送った要求 = %v、期待値 = 作成の1件", methods(requests))
	}
}

func TestCreate_CreatedWikilinks(t *testing.T) {
	t.Parallel()

	t.Run("JSONでも標準エラーに出し、標準出力を汚さない", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		var sent sentRequest
		handleCreate(t, e, &sent)
		handlePreview(t, e, previewLink("議事録", "existing_page"), previewLink("予算案", "new_page"))

		if err := e.Run(NewCmdPage, "page", "create", "--topic", "3", "--title", "メモ", "--body", "[[議事録]] [[予算案]]\n"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}

		var got map[string]any
		cmdtest.Decode(t, e.Stdout.String(), &got)
		if got["number"] != float64(12) || got["etag"] != etag {
			t.Errorf("標準出力 = %v、期待値 = 作成したページ", got)
		}
		if want := "存在しないリンク先1件を、未公開のページとして作成した見込みです: 開発/予算案\n"; e.Stderr.String() != want {
			t.Errorf("標準エラー = %q、期待値 = %q", e.Stderr.String(), want)
		}
	})

	t.Run("人間向けの表示では完了の行の後に出す", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		var sent sentRequest
		handleCreate(t, e, &sent)
		handlePreview(t, e, previewLink("予算案", "new_page"))

		if err := e.RunTTY(NewCmdPage, "page", "create", "--topic", "3", "--title", "メモ", "--body", "[[予算案]]\n"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		want := "ページ12を作成しました。\n存在しないリンク先1件を、未公開のページとして作成した見込みです: 開発/予算案\n"
		if e.Stderr.String() != want {
			t.Errorf("標準エラー = %q、期待値 = %q", e.Stderr.String(), want)
		}
	})
}

func TestCreate_WikilinkPreviewErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		status       int
		wantCode     string
		wantExitCode int
		wantHint     string
	}{
		{name: "トピックが無い", status: http.StatusUnprocessableEntity, wantCode: output.CodeValidationFailed, wantExitCode: output.ExitValidation, wantHint: "`wikino topic list`"},
		{name: "権限が無い", status: http.StatusForbidden, wantCode: output.CodeForbidden, wantExitCode: output.ExitForbidden},
		{name: "サーバーのエラー", status: http.StatusInternalServerError, wantCode: output.CodeServerError, wantExitCode: output.ExitError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodPost, previewPath, func(req *http.Request) *http.Response {
				return cmdtest.Problem(tt.status, "失敗しました。")
			})

			err := e.Run(NewCmdPage, "page", "create", "--topic", "3", "--title", "メモ", "--body", "[[予算案]]\n")
			cliErr := cmdtest.AssertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			// 書き込みの要求を送っていないことが分かるメッセージにする (作成済みかを確かめさせない)。
			if !strings.Contains(cliErr.Message, "書き込みの要求を送っていません") {
				t.Errorf("メッセージ %q に要求を送っていないことが無い", cliErr.Message)
			}
			if strings.Contains(cliErr.Hint, "wikino page list") {
				t.Errorf("作成していないのに作成済みかを確かめるヒントがある: %q", cliErr.Hint)
			}
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒント %q に %q が無い", cliErr.Hint, tt.wantHint)
			}
			if requests := e.Requests(); len(requests) != 1 {
				t.Errorf("送った要求 = %v、期待値 = プレビューの1件", methods(requests))
			}
		})
	}
}

func TestUpdate_WikilinkPreview(t *testing.T) {
	t.Parallel()

	t.Run("ページのトピックで確かめ、作られるリンク先を標準エラーに出す", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		var sent sentRequest
		handleUpdate(t, e, &sent)
		e.Handle(http.MethodGet, pagePath, func(req *http.Request) *http.Response {
			page := testPage(12, ptr("議事録"), "# 議事録\n")
			page["topic_number"] = 5
			return cmdtest.JSON(http.StatusOK, page, "ETag", etag)
		})
		preview := handlePreview(t, e, previewLink("予算案", "new_page"))

		const body = "# 議事録\n\n[[予算案]]\n"
		if err := e.Run(NewCmdPage, "page", "update", "12", "--if-match", etag, "--body", body); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}

		if got := preview.requests; len(got) != 1 || got[0]["topic_number"] != float64(5) || got[0]["body"] != body {
			t.Errorf("プレビューの要求 = %v、期待値 = トピック5と送る本文", got)
		}
		// 版は `--if-match` のものを送り、取得した版で置き換えない。
		if sent.ifMatch != etag {
			t.Errorf("If-Match = %q、期待値 = %q", sent.ifMatch, etag)
		}
		if want := "存在しないリンク先1件を、未公開のページとして作成した見込みです: 開発/予算案\n"; e.Stderr.String() != want {
			t.Errorf("標準エラー = %q、期待値 = %q", e.Stderr.String(), want)
		}
	})

	t.Run("タイトルだけの更新では確かめない", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		var sent sentRequest
		handleUpdate(t, e, &sent)

		if err := e.Run(NewCmdPage, "page", "update", "12", "--if-match", etag, "--title", "定例会議"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		if requests := e.Requests(); len(requests) != 1 || requests[0].Method != http.MethodPatch {
			t.Errorf("送った要求 = %v、期待値 = 更新の1件", methods(requests))
		}
	})

	t.Run("dry-runでリンクの解決の結果を出す", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		handleCurrentPage(e, etag)
		handlePreview(t, e, previewLink("予算案", "new_page"))

		if err := e.Run(NewCmdPage, "page", "update", "12", "--if-match", etag, "--body", "[[予算案]]\n", "--dry-run"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}

		var got wikilinkDryRunOutput
		cmdtest.Decode(t, e.Stdout.String(), &got)
		if len(got.Requests) != 1 || got.Requests[0].Method != http.MethodPatch {
			t.Errorf("送る要求 = %+v", got.Requests)
		}
		if len(got.Wikilinks) != 1 || got.Wikilinks[0].Resolution != "new_page" || got.Wikilinks[0].PageTitle != "予算案" {
			t.Errorf("wikilinks = %+v", got.Wikilinks)
		}
		if !strings.Contains(e.Stderr.String(), "書き込むと、存在しないリンク先1件") {
			t.Errorf("標準エラーに作られるリンク先が無い: %q", e.Stderr.String())
		}
	})
}
