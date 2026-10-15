package draft

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const (
	publicationPath = pagesPath + "/12/draft_page/publication"
	publishedETag   = `"published"`
)

// publishBodyは、公開する下書きの本文。今のページの本文 ("# 議事録\n") に3行を足し、存在しないページへのWikiリンクを2つ持つ。
const publishBody = "# 議事録\n\n- [[予算案]]\n- [[定例会議]]\n"

// handlePublishDraftは、ページ12の下書きの取得の要求に、版currentETagの下書きで応答する。
func handlePublishDraft(e *cmdtest.Env, title *string, currentETag string) {
	e.Handle(http.MethodGet, draftPath(12), func(req *http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testDraft(12, title, publishBody), "ETag", currentETag)
	})
}

// handlePublishedPageは、ページ12の取得の要求に、公開状態statusのページで応答する。確認で今のページと比べるために使う。
func handlePublishedPage(e *cmdtest.Env, status string) {
	e.Handle(http.MethodGet, pagesPath+"/12", func(req *http.Request) *http.Response {
		page := testPage(12, 3)
		page["status"] = status
		return cmdtest.JSON(http.StatusOK, page, "ETag", `"page"`)
	})
}

// handlePublishは、ページ12の下書きの公開の要求を受けて送った版をifMatchに記録し、公開したページを返す。
func handlePublish(e *cmdtest.Env, ifMatch *string) {
	e.Handle(http.MethodPost, publicationPath, func(req *http.Request) *http.Response {
		*ifMatch = req.Header.Get("If-Match")
		page := testPage(12, 3)
		page["title"] = "定例会議の議事録"
		page["body"] = publishBody
		return cmdtest.JSON(http.StatusOK, page, "ETag", publishedETag, "Content-Location", "/api/v1/spaces/team/pages/12")
	})
}

// assertNotPublishedは、公開の要求を送っていないことを確かめる。
func assertNotPublished(t *testing.T, e *cmdtest.Env) {
	t.Helper()

	for _, path := range requestPaths(e) {
		if path == http.MethodPost+" "+publicationPath {
			t.Errorf("公開の要求を送った: %s", path)
		}
	}
}

// assertPublishConfirmationは、公開の要求を送らずに終了コード9で確認プロトコルのJSONを出したことを確かめる。
func assertPublishConfirmation(t *testing.T, e *cmdtest.Env, err error) confirmationOutput {
	t.Helper()

	var silent *output.SilentError
	if !errors.As(err, &silent) || silent.ExitCode != output.ExitConfirmationRequired {
		t.Fatalf("終了コード9を期待したが %v だった", err)
	}
	assertNotPublished(t, e)

	var got confirmationOutput
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if got.Status != "confirmation_required" {
		t.Errorf("status = %q、期待値 = confirmation_required", got.Status)
	}
	assertNoSecret(t, e)

	return got
}

func TestPublish_ConfirmationRequired(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePublishDraft(e, ptr("定例会議の議事録"), etag)
	handlePublishedPage(e, "published")
	preview := handlePreview(t, e, newPageLink("予算案"), newPageLink("定例会議"))

	err := e.Run(NewCmdDraft, "draft", "publish", "12", "--if-match", etag, "--space", "team")
	got := assertPublishConfirmation(t, e, err)

	wantChanges := []string{
		"下書きの内容でページ12「定例会議の議事録」を公開します (タイトルを「議事録」から変更、本文 +3行 -0行)",
		"存在しないリンク先2件を未公開のページとして作成します: 開発/予算案, 開発/定例会議",
		"公開後、この下書きと下書きの編集履歴は削除されます",
	}
	if strings.Join(got.Changes, "\n") != strings.Join(wantChanges, "\n") {
		t.Errorf("changes = %q、期待値 = %q", got.Changes, wantChanges)
	}
	// 承認後のコマンドは、確認したときと同じスペースで、取得した版を固定して実行する。
	if want := `wikino draft publish 12 --if-match '"d41d8cd9"' --space team --yes`; got.ConfirmCommand != want {
		t.Errorf("confirm_command = %q、期待値 = %q", got.ConfirmCommand, want)
	}
	// リンク先は、公開する下書きの本文とトピックで確かめる。
	if sent := preview.sent(); len(sent) != 1 || sent[0]["body"] != publishBody || sent[0]["topic_number"] != float64(3) {
		t.Errorf("プレビューの要求 = %v", sent)
	}
}

func TestPublish_ConfirmationUnpublishedPage(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePublishDraft(e, ptr("議事録"), etag)
	handlePublishedPage(e, "unpublished")
	handlePreview(t, e)

	err := e.Run(NewCmdDraft, "draft", "publish", "12", "--if-match", etag)
	got := assertPublishConfirmation(t, e, err)

	// タイトルが同じで、作られるリンク先が無ければ、その行を出さない。
	wantChanges := []string{
		"下書きの内容でページ12「議事録」を公開します (タイトルの変更なし、本文 +3行 -0行)",
		"ページ12はまだ公開していないページで、公開済みになります",
		"公開後、この下書きと下書きの編集履歴は削除されます",
	}
	if strings.Join(got.Changes, "\n") != strings.Join(wantChanges, "\n") {
		t.Errorf("changes = %q、期待値 = %q", got.Changes, wantChanges)
	}
}

func TestPublish_ConfirmationWithForce(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePublishDraft(e, ptr("議事録"), `"latest"`)
	handlePublishedPage(e, "published")
	handlePreview(t, e)

	err := e.Run(NewCmdDraft, "draft", "publish", "12", "--force")
	got := assertPublishConfirmation(t, e, err)

	// `--force` で取得した版を固定し、`--force` は引き継がない。
	if want := `wikino draft publish 12 --if-match '"latest"' --yes`; got.ConfirmCommand != want {
		t.Errorf("confirm_command = %q、期待値 = %q", got.ConfirmCommand, want)
	}
}

func TestPublish_UntitledBeforeConfirmation(t *testing.T) {
	t.Parallel()

	// タイトルの無い下書きは公開できないため、承認を求めずに失敗させる。
	e := newEnv(t)
	handlePublishDraft(e, nil, etag)
	handlePreview(t, e)

	err := e.Run(NewCmdDraft, "draft", "publish", "12", "--if-match", etag)
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeValidationFailed, output.ExitValidation)
	if !strings.Contains(cliErr.Hint, "wikino draft update 12 --title") {
		t.Errorf("ヒントにタイトルを直すコマンドが無い: %q", cliErr.Hint)
	}
	assertNotPublished(t, e)
	if e.Stdout.Len() > 0 {
		t.Errorf("標準出力に確認が出た: %q", e.Stdout.String())
	}
}

func TestPublish_VersionChangedBeforeConfirmation(t *testing.T) {
	t.Parallel()

	// 読んだ版の後に下書きが更新されていれば、読んでいない内容の公開を確かめさせずに失敗させる。
	// `draft delete` と違い、確認を挟まない `--yes` と `--dry-run` でも、リンク先の確認や公開の要求の前に失敗させる。
	tests := []struct {
		name string
		args []string
	}{
		{name: "確認を求める"},
		{name: "--yes", args: []string{"--yes"}},
		{name: "--dry-run", args: []string{"--dry-run"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			handlePublishDraft(e, ptr("議事録"), `"latest"`)

			args := append([]string{"draft", "publish", "12", "--if-match", etag}, tt.args...)
			err := e.Run(NewCmdDraft, args...)
			cliErr := cmdtest.AssertCLIError(t, err, output.CodePreconditionFailed, output.ExitConflict)
			if !strings.Contains(cliErr.Hint, "`wikino draft get 12`") {
				t.Errorf("ヒントに今の内容を確かめるコマンドが無い: %q", cliErr.Hint)
			}
			if got := requestPaths(e); len(got) != 1 {
				t.Errorf("要求 = %v、期待値 = 下書きの取得の1件だけ", got)
			}
			if e.Stdout.Len() > 0 {
				t.Errorf("標準出力に出力があった: %q", e.Stdout.String())
			}
		})
	}
}

func TestPublish_Yes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantETag string
	}{
		{name: "--if-match", args: []string{"--if-match", `"latest"`}, wantETag: `"latest"`},
		{name: "--force", args: []string{"--force"}, wantETag: `"latest"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			handlePublishDraft(e, ptr("定例会議の議事録"), `"latest"`)
			handlePreview(t, e, newPageLink("予算案"))
			var ifMatch string
			handlePublish(e, &ifMatch)

			args := append([]string{"draft", "publish", "12", "--yes"}, tt.args...)
			if err := e.Run(NewCmdDraft, args...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			// `--yes` でも、公開する本文のリンク先を確かめるために下書きを取得する。確認のためのページの取得はしない。
			wantPaths := []string{"GET " + draftPath(12), "POST " + previewPath, "POST " + publicationPath}
			if got := requestPaths(e); strings.Join(got, "|") != strings.Join(wantPaths, "|") {
				t.Errorf("要求 = %v、期待値 = %v", got, wantPaths)
			}
			if ifMatch != tt.wantETag {
				t.Errorf("If-Match = %q、期待値 = %q", ifMatch, tt.wantETag)
			}

			// 公開したページを `page get` と同じ項目で出す。
			var got map[string]any
			cmdtest.Decode(t, e.Stdout.String(), &got)
			want := map[string]any{
				"id":           "00000000-0000-0000-0000-000000000012",
				"number":       float64(12),
				"title":        "定例会議の議事録",
				"status":       "published",
				"topic_number": float64(3),
				"modified_at":  "2026-10-01T09:00:00.123456Z",
				"url":          "https://wikino.example.com/s/team/pages/12",
				"body":         publishBody,
				"etag":         publishedETag,
			}
			if len(got) != len(want) {
				t.Errorf("出力 = %v、期待値 = %v", got, want)
			}
			for key, value := range want {
				if got[key] != value {
					t.Errorf("%s = %v、期待値 = %v", key, got[key], value)
				}
			}

			// 作られた見込みのリンク先は、標準出力を汚さないよう標準エラーに出す。
			if want := "存在しないリンク先1件を、未公開のページとして作成した見込みです: 開発/予算案\n"; e.Stderr.String() != want {
				t.Errorf("標準エラー = %q、期待値 = %q", e.Stderr.String(), want)
			}
		})
	}
}

func TestPublish_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePublishDraft(e, ptr("定例会議の議事録"), etag)
	handlePreview(t, e)
	var ifMatch string
	handlePublish(e, &ifMatch)

	if err := e.RunTTY(NewCmdDraft, "draft", "publish", "12", "--if-match", etag, "--yes"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "publish", e.Stdout.String())
	if got, want := e.Stderr.String(), "ページ12を下書きの内容で公開しました。\n"; got != want {
		t.Errorf("標準エラー = %q、期待値 = %q", got, want)
	}
}

func TestPublish_Prompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		answer      string
		wantPublish bool
	}{
		{name: "承認", answer: "y\n", wantPublish: true},
		{name: "拒否", answer: "n\n"},
		{name: "入力なし", answer: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Stdin = tt.answer
			handlePublishDraft(e, ptr("議事録"), etag)
			handlePublishedPage(e, "published")
			handlePreview(t, e, newPageLink("予算案"))
			var ifMatch string
			handlePublish(e, &ifMatch)

			err := e.RunTTY(NewCmdDraft, "draft", "publish", "12", "--if-match", etag)

			// 端末では、標準エラーに公開する内容を示して尋ねる。
			for _, want := range []string{"- 下書きの内容でページ12「議事録」を公開します", "開発/予算案", "下書きの編集履歴は削除されます"} {
				if !strings.Contains(e.Stderr.String(), want) {
					t.Errorf("標準エラーに %q が無い: %q", want, e.Stderr.String())
				}
			}
			if tt.wantPublish {
				if err != nil {
					t.Fatalf("エラーを期待しなかったが %v だった", err)
				}
				if ifMatch != etag {
					t.Errorf("If-Match = %q、期待値 = %q", ifMatch, etag)
				}
				return
			}

			_ = cmdtest.AssertCLIError(t, err, output.CodeConfirmationDeclined, output.ExitConfirmationRequired)
			assertNotPublished(t, e)
		})
	}
}

func TestPublish_DryRun(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	handlePublishDraft(e, ptr("議事録"), etag)
	handlePreview(t, e, newPageLink("予算案"))

	if err := e.Run(NewCmdDraft, "draft", "publish", "12", "--if-match", etag, "--dry-run"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// `--dry-run` は何も公開しないため、確認を求めない。下書きの取得とリンク先の確認 (読み取り) だけを送る。
	wantPaths := []string{"GET " + draftPath(12), "POST " + previewPath}
	if got := requestPaths(e); strings.Join(got, "|") != strings.Join(wantPaths, "|") {
		t.Errorf("要求 = %v、期待値 = %v", got, wantPaths)
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
	if req := got.Requests[0]; req.Method != http.MethodPost || req.URL != cmdtest.Host+publicationPath || req.Headers["If-Match"] != etag {
		t.Errorf("記録した要求 = %+v", req)
	}
	if len(got.Wikilinks) != 1 || got.Wikilinks[0]["page_title"] != "予算案" {
		t.Errorf("wikilinks = %v", got.Wikilinks)
	}
	if !strings.Contains(e.Stderr.String(), "開発/予算案") {
		t.Errorf("標準エラーに作られるリンク先が無い: %q", e.Stderr.String())
	}
	assertNoSecret(t, e)
}

func TestPublish_APIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		title        *string
		status       int
		wantCode     string
		wantExitCode int
		wantHint     string
	}{
		{
			// `--yes` では確認の前の検査を通らないため、タイトルの無い下書きの公開はAPIの422になる。
			name:         "タイトルの無い下書きの422",
			status:       http.StatusUnprocessableEntity,
			wantCode:     output.CodeValidationFailed,
			wantExitCode: output.ExitValidation,
			wantHint:     "`wikino draft update 12 --title '<title>' --if-match '<etag>'`",
		},
		{name: "412", title: ptr("議事録"), status: http.StatusPreconditionFailed, wantCode: output.CodePreconditionFailed, wantExitCode: output.ExitConflict, wantHint: "`wikino draft get 12`"},
		{name: "404", title: ptr("議事録"), status: http.StatusNotFound, wantCode: output.CodeNotFound, wantExitCode: output.ExitNotFound, wantHint: "`wikino page get 12`"},
		{name: "428", title: ptr("議事録"), status: http.StatusPreconditionRequired, wantCode: output.CodePreconditionRequired, wantExitCode: output.ExitConflict, wantHint: "`wikino draft get 12`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			handlePublishDraft(e, tt.title, etag)
			handlePreview(t, e)
			e.Handle(http.MethodPost, publicationPath, func(req *http.Request) *http.Response {
				return cmdtest.Problem(tt.status, "失敗しました。")
			})

			err := e.Run(NewCmdDraft, "draft", "publish", "12", "--if-match", etag, "--yes")
			cliErr := cmdtest.AssertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒント %q に %q が無い", cliErr.Hint, tt.wantHint)
			}
		})
	}
}

func TestPublish_DraftNotFound(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, draftPath(12), func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "下書きが見つかりません。")
	})

	err := e.Run(NewCmdDraft, "draft", "publish", "12", "--if-match", etag, "--yes")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "wikino draft list") {
		t.Errorf("ヒントに下書きの一覧のコマンドが無い: %q", cliErr.Hint)
	}
	assertNotPublished(t, e)
}

func TestPublish_UsageErrors(t *testing.T) {
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
			args:        []string{"0", "--if-match", etag},
			wantMessage: "<page_number>",
		},
		{
			name:        "--dry-runと--fieldsの併用",
			args:        []string{"12", "--if-match", etag, "--dry-run", "--fields", "number"},
			wantMessage: "--dry-run, --fields",
		},
		{
			name:        "未知の項目",
			args:        []string{"12", "--if-match", etag, "--fields", "page_number"},
			wantMessage: "page_number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdDraft, append([]string{"draft", "publish"}, tt.args...)...)
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
