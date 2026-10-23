package attachment

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const (
	attachmentID   = "01234567-89ab-cdef-0123-456789abcdef"
	attachmentPath = "/api/v1/spaces/team/attachments/" + attachmentID
	// downloadURLは、テストの署名付きURL。クエリの署名は出力やエラーに出してはいけない値。
	downloadURL    = "https://storage.example.com/blobs/key"
	downloadSecret = "X-Amz-Signature=secret-signature"
	content        = "\x89PNG\r\n\x1a\nimage-data"
)

func testAttachment(filename string) map[string]any {
	return map[string]any{
		"id":                      attachmentID,
		"filename":                filename,
		"content_type":            "image/png",
		"byte_size":               len(content),
		"download_url":            downloadURL + "?" + downloadSecret,
		"download_url_expires_at": "2026-10-15T09:05:00Z",
	}
}

// handleAttachmentは、filenameの添付ファイルの情報と、署名付きURLの中身を返すスタブを登録する。
func handleAttachment(e *cmdtest.Env, filename string) {
	e.Handle(http.MethodGet, attachmentPath, func(*http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testAttachment(filename))
	})
	e.HandleURL(http.MethodGet, downloadURL, func(*http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"image/png"}},
			Body:       io.NopCloser(strings.NewReader(content)),
		}
	})
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path) // #nosec G304 -- テストが作った一時ディレクトリのファイル
	if err != nil {
		t.Fatalf("保存したファイルを読めない: %v", err)
	}

	return string(b)
}

func TestGet_JSON(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleAttachment(e, "diagram.png")
	dir := t.TempDir()

	if err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "--output", dir); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	path := filepath.Join(dir, "diagram.png")
	var got map[string]any
	cmdtest.Decode(t, e.Stdout.String(), &got)
	want := map[string]any{
		"id":           attachmentID,
		"path":         path,
		"filename":     "diagram.png",
		"content_type": "image/png",
		"byte_size":    float64(len(content)),
	}
	if len(got) != len(want) {
		t.Errorf("出力の項目 = %v、期待値 = %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v、期待値 = %v", k, got[k], v)
		}
	}
	if got := readFile(t, path); got != content {
		t.Errorf("保存した中身 = %q、期待値 = %q", got, content)
	}

	requests := e.Requests()
	if len(requests) != 2 {
		t.Fatalf("要求の数 = %d、期待値 = 2", len(requests))
	}
	if got := requests[1].Header.Get("Authorization"); got != "" {
		t.Errorf("署名付きURLへの要求にAuthorizationが付いている: %q", got)
	}
	if got := requests[1].URL.RawQuery; got != downloadSecret {
		t.Errorf("署名付きURLのクエリ = %q、期待値 = %q", got, downloadSecret)
	}
	if out := e.Stdout.String() + e.Stderr.String(); strings.Contains(out, "secret-signature") {
		t.Errorf("出力に署名付きURLが含まれている: %q", out)
	}
}

// カレントディレクトリを変えるため、並列に実行しない。
func TestGet_HumanToCurrentDirectory(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Chdir(dir)

	e := cmdtest.New(t)
	handleAttachment(e, "diagram.png")

	if err := e.RunTTY(NewCmdAttachment, "attachment", "get", attachmentID); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	// ゴールデンファイルはパッケージのディレクトリから読むため、戻してから比べる。
	t.Chdir(wd)
	cmdtest.AssertGolden(t, "get", e.Stdout.String())
	if got := readFile(t, filepath.Join(dir, "diagram.png")); got != content {
		t.Errorf("保存した中身 = %q、期待値 = %q", got, content)
	}
}

func TestGet_ReferencePath(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleAttachment(e, "diagram.png")
	dir := t.TempDir()

	if err := e.Run(NewCmdAttachment, "attachment", "get", "/attachments/"+attachmentID, "-O", dir); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if got := readFile(t, filepath.Join(dir, "diagram.png")); got != content {
		t.Errorf("保存した中身 = %q、期待値 = %q", got, content)
	}
}

func TestGet_InvalidID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  string
	}{
		{name: "UUIDでない", arg: "12"},
		{name: "パスの階層を移る", arg: "../" + attachmentID},
		{name: "参照の後ろにクエリ", arg: "/attachments/" + attachmentID + "?x=1"},
		{name: "URL", arg: "https://wikino.example.com/attachments/" + attachmentID},
		{name: "改行の混入", arg: attachmentID + "\n"},
		{name: "波括弧付きのUUID", arg: "{" + attachmentID + "}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := cmdtest.New(t)
			err := e.Run(NewCmdAttachment, "attachment", "get", tt.arg, "-O", t.TempDir())

			_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if n := len(e.Requests()); n != 0 {
				t.Errorf("要求を送らないことを期待したが %d件送った", n)
			}
		})
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	e.Handle(http.MethodGet, attachmentPath, func(*http.Request) *http.Response {
		return cmdtest.Problem(http.StatusNotFound, "添付ファイルが見つかりません。")
	})

	err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", t.TempDir())

	cliErr := cmdtest.AssertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
	if !strings.Contains(cliErr.Hint, "/attachments/{ID}") {
		t.Errorf("ヒントに添付ファイルのIDの確かめ方が無い: %q", cliErr.Hint)
	}
}

func TestGet_UnsafeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "親の階層を指す", filename: "../../escape.png", want: "_.._escape.png"},
		{name: "絶対パス", filename: "/tmp/escape.png", want: "_tmp_escape.png"},
		{name: "Windowsのパスの区切り", filename: `..\..\escape.png`, want: "_.._escape.png"},
		{name: "制御文字", filename: "esc\nape\x1b.png", want: "escape.png"},
		{name: "..だけ", filename: "..", want: attachmentID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := cmdtest.New(t)
			handleAttachment(e, tt.filename)
			// 保存先のディレクトリの外に書かれていないことを確かめるため、親のディレクトリを空で用意する。
			parent := t.TempDir()
			dir := filepath.Join(parent, "a", "b")
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Fatal(err)
			}

			if err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", dir); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != tt.want {
				t.Errorf("保存したファイル = %v、期待値 = %q", entries, tt.want)
			}
			for _, d := range []string{parent, filepath.Join(parent, "a")} {
				if entries, _ := os.ReadDir(d); len(entries) != 1 {
					t.Errorf("保存先のディレクトリの外 %s にファイルがある: %v", d, entries)
				}
			}
		})
	}
}

func TestGet_ExistingFile(t *testing.T) {
	t.Parallel()

	t.Run("保存先のファイルを指定したときは要求を送らずに失敗する", func(t *testing.T) {
		t.Parallel()

		e := cmdtest.New(t)
		path := filepath.Join(t.TempDir(), "saved.png")
		if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}

		err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", path)

		cliErr := cmdtest.AssertCLIError(t, err, output.CodeFileExists, output.ExitUsage)
		if !strings.Contains(cliErr.Hint, "--clobber") {
			t.Errorf("ヒントに `--clobber` が無い: %q", cliErr.Hint)
		}
		if n := len(e.Requests()); n != 0 {
			t.Errorf("要求を送らないことを期待したが %d件送った", n)
		}
		if got := readFile(t, path); got != "existing" {
			t.Errorf("既存のファイルが書き換わった: %q", got)
		}
	})

	t.Run("元のファイル名で保存するときは中身を取得せずに失敗する", func(t *testing.T) {
		t.Parallel()

		e := cmdtest.New(t)
		handleAttachment(e, "diagram.png")
		dir := t.TempDir()
		path := filepath.Join(dir, "diagram.png")
		if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}

		err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", dir)

		_ = cmdtest.AssertCLIError(t, err, output.CodeFileExists, output.ExitUsage)
		if n := len(e.Requests()); n != 1 {
			t.Errorf("添付ファイルの情報の取得だけを期待したが %d件送った", n)
		}
		if got := readFile(t, path); got != "existing" {
			t.Errorf("既存のファイルが書き換わった: %q", got)
		}
	})

	t.Run("壊れたシンボリックリンクも上書きしない", func(t *testing.T) {
		t.Parallel()

		e := cmdtest.New(t)
		dir := t.TempDir()
		target := filepath.Join(dir, "target")
		path := filepath.Join(dir, "link.png")
		if err := os.Symlink(target, path); err != nil {
			t.Skipf("シンボリックリンクを作れない: %v", err)
		}

		err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", path)

		_ = cmdtest.AssertCLIError(t, err, output.CodeFileExists, output.ExitUsage)
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Errorf("リンク先にファイルが作られた: %v", err)
		}
	})

	t.Run("--clobberなら上書きする", func(t *testing.T) {
		t.Parallel()

		e := cmdtest.New(t)
		handleAttachment(e, "diagram.png")
		dir := t.TempDir()
		path := filepath.Join(dir, "diagram.png")
		if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}

		if err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", dir, "--clobber"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		if got := readFile(t, path); got != content {
			t.Errorf("保存した中身 = %q、期待値 = %q", got, content)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 1 {
			t.Errorf("一時ファイルが残っている: %v", entries)
		}
	})
}

func TestGet_Stdout(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleAttachment(e, "diagram.png")
	t.Run("中身だけを標準出力に出す", func(t *testing.T) {
		if err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", "-"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		if got := e.Stdout.String(); got != content {
			t.Errorf("標準出力 = %q、期待値 = %q", got, content)
		}
		if got := e.Stderr.String(); got != "" {
			t.Errorf("標準エラー = %q、期待値は空", got)
		}
	})

	t.Run("標準出力が端末なら要求を送らずに使い方の誤りにする", func(t *testing.T) {
		e := cmdtest.New(t)

		err := e.RunTTY(NewCmdAttachment, "attachment", "get", attachmentID, "-O", "-")

		cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
		if !strings.Contains(cliErr.Hint, "-O <path>") {
			t.Errorf("ヒントにファイルへの保存の案内が無い: %q", cliErr.Hint)
		}
		if n := len(e.Requests()); n != 0 {
			t.Errorf("要求を送らないことを期待したが %d件送った", n)
		}
		if got := e.Stdout.String(); got != "" {
			t.Errorf("標準出力 = %q、期待値は空", got)
		}
	})

	for _, flag := range []string{"--json", "--jq=.id", "--fields=id", "--clobber"} {
		t.Run(flag+"とは併用できない", func(t *testing.T) {
			e := cmdtest.New(t)

			err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", "-", flag)

			_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if n := len(e.Requests()); n != 0 {
				t.Errorf("要求を送らないことを期待したが %d件送った", n)
			}
		})
	}
}

func TestGet_DownloadFailed(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	e.Handle(http.MethodGet, attachmentPath, func(*http.Request) *http.Response {
		return cmdtest.JSON(http.StatusOK, testAttachment("diagram.png"))
	})
	e.HandleURL(http.MethodGet, downloadURL, func(*http.Request) *http.Response {
		// オブジェクトストレージは、期限切れの署名付きURLに403をXMLで返す。
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"Content-Type": {"application/xml"}},
			Body:       io.NopCloser(strings.NewReader("<Error><Code>AccessDenied</Code><Message>Request has expired</Message></Error>")),
		}
	})
	dir := t.TempDir()

	err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", dir)

	cliErr := cmdtest.AssertCLIError(t, err, output.CodeDownloadFailed, output.ExitError)
	if cliErr.Status != http.StatusForbidden {
		t.Errorf("status = %d、期待値 = 403", cliErr.Status)
	}
	if msg := cliErr.Message + cliErr.Hint; strings.Contains(msg, "secret-signature") {
		t.Errorf("エラーに署名付きURLが含まれている: %q", msg)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("取得に失敗したのにファイルが残っている: %v", entries)
	}
}

func TestGet_InsecureDownloadURL(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	e.Handle(http.MethodGet, attachmentPath, func(*http.Request) *http.Response {
		attachment := testAttachment("diagram.png")
		attachment["download_url"] = "http://storage.example.com/blobs/key?" + downloadSecret
		return cmdtest.JSON(http.StatusOK, attachment)
	})

	err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", t.TempDir())

	cliErr := cmdtest.AssertCLIError(t, err, output.CodeResponse, output.ExitError)
	if strings.Contains(cliErr.Message, "secret-signature") {
		t.Errorf("エラーに署名付きURLが含まれている: %q", cliErr.Message)
	}
	if n := len(e.Requests()); n != 1 {
		t.Errorf("署名付きURLへ要求を送らないことを期待したが、要求は %d件だった", n)
	}
}

func TestGet_UnresolvedRedirect(t *testing.T) {
	t.Parallel()
	for _, response := range []struct {
		name   string
		status int
	}{
		{name: "Locationの無い302", status: http.StatusFound},
		{name: "304", status: http.StatusNotModified},
	} {
		t.Run(response.name, func(t *testing.T) {
			t.Parallel()
			for _, mode := range []struct {
				name    string
				clobber bool
			}{
				{name: "新規保存"}, {name: "既存ファイルの置き換え", clobber: true},
			} {
				t.Run(mode.name, func(t *testing.T) {
					t.Parallel()
					e := cmdtest.New(t)
					handleAttachment(e, "diagram.png")
					e.HandleURL(http.MethodGet, downloadURL, func(*http.Request) *http.Response {
						return &http.Response{StatusCode: response.status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("redirect document"))}
					})
					dir := t.TempDir()
					path := filepath.Join(dir, "diagram.png")
					args := []string{"attachment", "get", attachmentID, "-O", dir}
					if mode.clobber {
						if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
							t.Fatal(err)
						}
						args = append(args, "--clobber")
					}
					err := e.Run(NewCmdAttachment, args...)
					cliErr := cmdtest.AssertCLIError(t, err, output.CodeDownloadFailed, output.ExitError)
					if cliErr.Status != response.status {
						t.Errorf("ステータス = %d、期待値 = %d", cliErr.Status, response.status)
					}
					if e.Stdout.Len() != 0 {
						t.Errorf("成功結果が出力されました: %s", e.Stdout.String())
					}
					entries, err := os.ReadDir(dir)
					if err != nil {
						t.Fatal(err)
					}
					if mode.clobber {
						if got := readFile(t, path); got != "existing" {
							t.Errorf("既存ファイルが変更されました: %q", got)
						}
						if len(entries) != 1 {
							t.Errorf("不要なファイルが残っています: %v", entries)
						}
					} else if len(entries) != 0 {
						t.Errorf("失敗したのにファイルが作られました: %v", entries)
					}
				})
			}
		})
	}
}

// attachmentReadErrorは、本文の一部を読んだ後の通信切断を再現するために使う。
type attachmentReadError struct{}

func (attachmentReadError) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestGet_InterruptedDownload(t *testing.T) {
	t.Parallel()
	for _, mode := range []struct {
		name    string
		clobber bool
	}{
		{name: "新規保存"}, {name: "既存ファイルの置き換え", clobber: true},
	} {
		t.Run(mode.name, func(t *testing.T) {
			t.Parallel()
			e := cmdtest.New(t)
			handleAttachment(e, "diagram.png")
			e.HandleURL(http.MethodGet, downloadURL, func(*http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK, Header: http.Header{},
					Body: io.NopCloser(io.MultiReader(strings.NewReader(content[:4]), attachmentReadError{})),
				}
			})
			dir := t.TempDir()
			path := filepath.Join(dir, "diagram.png")
			args := []string{"attachment", "get", attachmentID, "-O", dir}
			if mode.clobber {
				if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--clobber")
			}
			err := e.Run(NewCmdAttachment, args...)
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("本文の取得途中の失敗を期待しました: %v", err)
			}
			// 取得途中の切断は、署名付きURLへの通信の失敗として再実行を案内する。
			cliErr := cmdtest.AssertCLIError(t, err, output.CodeNetwork, output.ExitError)
			loc := i18n.NewLocalizer(i18n.LangJa)
			if cliErr.Message != loc.T("error_download_network") || cliErr.Hint != loc.T("error_download_failed_hint") {
				t.Errorf("翻訳された通信の失敗の案内になりません: %+v", cliErr)
			}
			if e.Stdout.Len() != 0 {
				t.Errorf("成功結果が出力されました: %s", e.Stdout.String())
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if mode.clobber {
				if got := readFile(t, path); got != "existing" {
					t.Errorf("既存ファイルが変更されました: %q", got)
				}
				if len(entries) != 1 {
					t.Errorf("一時ファイルが残っています: %v", entries)
				}
			} else if len(entries) != 0 {
				t.Errorf("書きかけのファイルが残っています: %v", entries)
			}
		})
	}
}

func TestGet_StdoutInterruptedDownload(t *testing.T) {
	t.Parallel()

	e := cmdtest.New(t)
	handleAttachment(e, "diagram.png")
	e.HandleURL(http.MethodGet, downloadURL, func(*http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK, Header: http.Header{},
			Body: io.NopCloser(io.MultiReader(strings.NewReader(content[:4]), attachmentReadError{})),
		}
	})

	err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", "-")

	_ = cmdtest.AssertCLIError(t, err, output.CodeNetwork, output.ExitError)
}

func TestGet_MissingDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name    string
		output  string
		wantDir string
	}{
		{name: "区切りで終わる無いディレクトリ", output: filepath.Join(dir, "missing") + "/", wantDir: filepath.Join(dir, "missing") + "/"},
		{name: "親の無いファイルのパス", output: filepath.Join(dir, "missing", "diagram.png"), wantDir: filepath.Join(dir, "missing")},
		{name: "区切りで終わるファイル", output: file + "/", wantDir: file + "/"},
		{name: "親がファイルのパス", output: filepath.Join(file, "diagram.png"), wantDir: file},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := cmdtest.New(t)

			err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", tt.output)

			cliErr := cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, tt.wantDir) {
				t.Errorf("メッセージに保存先のディレクトリ %q が無い: %q", tt.wantDir, cliErr.Message)
			}
			if n := len(e.Requests()); n != 0 {
				t.Errorf("要求を送らないことを期待したが %d件送った", n)
			}
		})
	}
}

func TestGet_WriteFailed(t *testing.T) {
	t.Parallel()

	// 元のファイル名と同じ名前のディレクトリがあると、`--clobber` の置き換えが失敗する。
	e := cmdtest.New(t)
	handleAttachment(e, "diagram.png")
	dir := t.TempDir()
	path := filepath.Join(dir, "diagram.png")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}

	err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", dir, "--clobber")

	cliErr := cmdtest.AssertCLIError(t, err, output.CodeWriteFailed, output.ExitError)
	if !strings.Contains(cliErr.Message, path) {
		t.Errorf("メッセージに保存先 %q が無い: %q", path, cliErr.Message)
	}
	if cliErr.Hint == "" {
		t.Error("ヒントが無い")
	}
	if e.Stdout.Len() != 0 {
		t.Errorf("成功結果が出力されました: %s", e.Stdout.String())
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("一時ファイルが残っている: %v", entries)
	}
}

// umaskPermは、umaskに従って作ったファイルの権限を返す。テストはumaskを変えずに期待値を求める。
func umaskPerm(t *testing.T) os.FileMode {
	t.Helper()

	path := filepath.Join(t.TempDir(), "reference")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	return filePermOf(t, path)
}

func filePermOf(t *testing.T, path string) os.FileMode {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	return info.Mode().Perm()
}

func TestGet_FilePermission(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windowsはファイルの権限のビットを持たない")
	}

	want := umaskPerm(t)

	t.Run("新規保存はumaskに従う", func(t *testing.T) {
		t.Parallel()

		e := cmdtest.New(t)
		handleAttachment(e, "diagram.png")
		dir := t.TempDir()

		if err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", dir); err != nil {
			t.Fatal(err)
		}
		if got := filePermOf(t, filepath.Join(dir, "diagram.png")); got != want {
			t.Errorf("権限 = %v、期待値 = %v", got, want)
		}
	})

	t.Run("--clobberで無いファイルを作るときもumaskに従う", func(t *testing.T) {
		t.Parallel()

		e := cmdtest.New(t)
		handleAttachment(e, "diagram.png")
		dir := t.TempDir()

		if err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", dir, "--clobber"); err != nil {
			t.Fatal(err)
		}
		if got := filePermOf(t, filepath.Join(dir, "diagram.png")); got != want {
			t.Errorf("権限 = %v、期待値 = %v", got, want)
		}
	})

	t.Run("--clobberは置き換えるファイルの権限を保つ", func(t *testing.T) {
		t.Parallel()

		e := cmdtest.New(t)
		handleAttachment(e, "diagram.png")
		path := filepath.Join(t.TempDir(), "diagram.png")
		if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}
		// umaskの影響を受けないよう、作った後に権限を変える。
		if err := os.Chmod(path, 0o640); err != nil {
			t.Fatal(err)
		}

		if err := e.Run(NewCmdAttachment, "attachment", "get", attachmentID, "-O", path, "--clobber"); err != nil {
			t.Fatal(err)
		}
		if got := readFile(t, path); got != content {
			t.Errorf("保存した中身 = %q、期待値 = %q", got, content)
		}
		if got := filePermOf(t, path); got != 0o640 {
			t.Errorf("権限 = %v、期待値 = %v", got, os.FileMode(0o640))
		}
	})
}
