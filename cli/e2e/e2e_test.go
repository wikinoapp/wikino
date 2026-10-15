//go:build e2e

// Package e2eは、ビルドしたCLIを開発用のサーバーに対して実行し、主要な流れを通すテスト。
//
// コマンドのテスト (internal/cmd) はAPIをスタブで受けるため、サーバーの実際の応答との
// ずれ (OpenAPI記述に無い振る舞い・認可・ETagの計算など) はここで検出する。
// ビルドタグ `e2e` を付けたときだけ動き、接続先は次の環境変数で受け取る。
// 準備 (トークンの発行) と実行はscripts/e2e.shが行う (make e2e)。
//
//   - WIKINO_E2E_HOST: 開発用のサーバーのURL (http://localhost:4201など)
//   - WIKINO_E2E_SPACE: シードのスペースの識別子
//   - WIKINO_E2E_TOKEN: そのスペースの個人アクセストークン
//
// CLIの環境変数 (WIKINO_TOKENなど) を直接読まないのは、手元で本番のトークンを
// 設定したままテストを走らせても、本番のスペースにページを作らないため。
package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// sandboxTopicNameは、ページと下書きを作るトピックの名前。シードの公開トピック
// (go/internal/seed/topic.goのtopicNameSandbox) で、シードの管理者が参加している。
const sandboxTopicName = "サンドボックス"

var (
	// binaryはTestMainがビルドしたCLIのパス。
	binary string
	// targetはCLIに渡す接続先・スペース・トークン。
	target struct {
		host, space, token string
	}
)

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

// runMainはCLIをビルドしてからテストを実行する。deferで一時ディレクトリを消すため、
// os.Exitを呼ぶTestMainと分ける。
func runMain(m *testing.M) int {
	target.host = os.Getenv("WIKINO_E2E_HOST")
	target.space = os.Getenv("WIKINO_E2E_SPACE")
	target.token = os.Getenv("WIKINO_E2E_TOKEN")
	if target.host == "" || target.space == "" || target.token == "" {
		// タグを付けて明示的に走らせたテストのため、スキップせず失敗させる。
		fmt.Fprintln(os.Stderr, "WIKINO_E2E_HOST・WIKINO_E2E_SPACE・WIKINO_E2E_TOKENを設定してください (make e2e を参照)")
		return 1
	}

	dir, err := os.MkdirTemp("", "wikino-e2e-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "一時ディレクトリを作れません: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()

	binary = filepath.Join(dir, "wikino")
	build := exec.Command("go", "build", "-o", binary, "./cmd/wikino")
	build.Dir = ".."
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "CLIのビルドに失敗しました: %v\n", err)
		return 1
	}

	return m.Run()
}

// resultはCLIを1回実行した結果。
type result struct {
	stdout, stderr string
	code           int
}

// runCLIはargsでCLIを実行する。標準出力はパイプのため、CLIは結果をJSONで出す。
// stdinが空でなければ標準入力に渡す。
func runCLI(t *testing.T, stdin string, args ...string) result {
	t.Helper()

	cmd := exec.Command(binary, args...)
	// 手元の設定・キーチェーン・表示言語に左右されないよう、環境変数は渡すものだけにする。
	cmd.Env = []string{
		"HOME=" + t.TempDir(),
		"XDG_CONFIG_HOME=" + t.TempDir(),
		"WIKINO_TOKEN_STORE=file",
		"WIKINO_LANG=ja",
		"WIKINO_HOST=" + target.host,
		"WIKINO_SPACE=" + target.space,
		"WIKINO_TOKEN=" + target.token,
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err := cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("wikino %s を実行できません: %v", strings.Join(args, " "), err)
	}

	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// mustRunはargsでCLIを実行し、終了コード0を確かめて標準出力を返す。
func mustRun(t *testing.T, stdin string, args ...string) string {
	t.Helper()

	r := runCLI(t, stdin, args...)
	if r.code != 0 {
		t.Fatalf("wikino %s の終了コード = %d、期待値 = 0\n標準出力: %s\n標準エラー: %s", strings.Join(args, " "), r.code, r.stdout, r.stderr)
	}

	return r.stdout
}

// mustRunJSONはmustRunの標準出力をvに読み込む。
func mustRunJSON(t *testing.T, v any, stdin string, args ...string) {
	t.Helper()

	decode(t, mustRun(t, stdin, args...), v)
}

// decodeはCLIの標準出力のJSONをvに読み込む。
func decode(t *testing.T, out string, v any) {
	t.Helper()

	if err := json.Unmarshal([]byte(out), v); err != nil {
		t.Fatalf("出力をJSONとして読めません: %v\n%s", err, out)
	}
}

// expectExitはargsでCLIを実行し、終了コードがwantであることを確かめる。
func expectExit(t *testing.T, want int, stdin string, args ...string) result {
	t.Helper()

	r := runCLI(t, stdin, args...)
	if r.code != want {
		t.Fatalf("wikino %s の終了コード = %d、期待値 = %d\n標準出力: %s\n標準エラー: %s", strings.Join(args, " "), r.code, want, r.stdout, r.stderr)
	}

	return r
}

// pageはページと下書きの出力のうち、テストで見る項目。
type page struct {
	ID         string  `json:"id"`
	Number     int     `json:"number"`
	PageNumber int     `json:"page_number"`
	Title      *string `json:"title"`
	Body       string  `json:"body"`
	Status     string  `json:"status"`
	Etag       string  `json:"etag"`
}

// listは一覧の出力。
type list struct {
	Items []page `json:"items"`
}

// uniqueWordは、テストが作るページのタイトルに入れる語。開発用のデータベースに
// 前回の実行のページが残っていても、タイトルが重ならず、検索でこの実行のページだけに
// 一致するようにする。検索の語にそのまま使えるよう空白を含めない。
func uniqueWord() string {
	return fmt.Sprintf("e2e%d", time.Now().UnixNano())
}

// sandboxTopicNumberは、ページを作るトピックの番号を返す。
func sandboxTopicNumber(t *testing.T) int {
	t.Helper()

	var topics struct {
		Items []struct {
			Number int    `json:"number"`
			Name   string `json:"name"`
		} `json:"items"`
	}
	mustRunJSON(t, &topics, "", "topic", "list", "--limit", "100")
	for _, topic := range topics.Items {
		if topic.Name == sandboxTopicName {
			return topic.Number
		}
	}
	t.Fatalf("トピック「%s」がありません。シードを投入したスペースか確かめてください", sandboxTopicName)

	return 0
}
