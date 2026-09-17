package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// browseLoginHarnessはbrowse.shをsourceして関数を取り込み、プロセスの外へ
// 出るもの (Goツールチェイン、playwright-cli、ベースURLが要るBasic認証configの
// 生成) だけを差し替える。テストの対象そのもの (役割の読み替え、2行の資格情報解析、
// 生成されるパスワードスクリプト、2要素認証分岐) は書かれたまま動く。
//
// ファイルパスはここに書き写さない。browse.shがWIKINO_BROWSE_TMP_DIRから取るため、
// 定義はbrowse.shのものだけになり、あちらで改名したときに、このハーネスだけが別の
// 場所を指したまま、スクリプトが開発者の実tmpディレクトリへパスワードのファイルを
// 書く、という状態にならない。
const browseLoginHarness = `
set -euo pipefail

source "$WIKINO_BROWSE_TEST_SCRIPT"

test_password="$WIKINO_BROWSE_TEST_PASSWORD"
test_email="$WIKINO_BROWSE_TEST_EMAIL"
unset WIKINO_BROWSE_TEST_PASSWORD WIKINO_BROWSE_TEST_EMAIL

go() {
  if [[ "$#" -eq 4 && "$1" == "run" && "$2" == "./cmd/wikino" && "$3" == "devcreds" ]]; then
    printf '%s' "$4" > "$WIKINO_BROWSE_TEST_CAPTURE/role"
    printf '%s
%s
' "$test_email" "$test_password"
    return 0
  fi
  if [[ "$#" -eq 2 && "$1" == "run" && "$2" == "./cmd/devtotp" ]]; then
    printf '%s' "${WIKINO_DEVTOTP_EMAIL:-}" > "$WIKINO_BROWSE_TEST_CAPTURE/devtotp-email"
    printf '123456
'
    return 0
  fi

  printf 'unexpected go command:' >&2
  printf ' %q' "$@" >&2
  printf '
' >&2
  return 64
}

build_config() {
  mkdir -p "$TMP_DIR"
  printf '{}
' > "$CONFIG_FILE"
  chmod 600 "$CONFIG_FILE"
  printf '%s' 'https://example.test' > "$ORIGIN_FILE"
}

# pwはpw_checkedではなくplaywright-cli自体の代役。どちらのラッパーも書かれた
# まま動かし、ここで捕捉するargvを、実際のコマンドが渡されたはずのargvにするため。
#
# run-codeにはplaywright-cliと同じ形で答える。実行したコードを返す形であり、
# サインインではそれが生成スクリプトと、その中のパスワードになる。これが
# pw_checked_secretの存在理由である応答そのものになる。実際の応答がその周りに置く
# jsのコードフェンスは省いた。Goのraw string literalはバッククォートを持てないため。
# フィルタが手掛かりにするのは "### " のセクション見出しで、そちらは再現している。
pw() {
  local arg
  {
    printf 'playwright-cli'
    for arg in "$@"; do
      printf '	%s' "$arg"
    done
    printf '
'
  } >> "$WIKINO_BROWSE_TEST_CAPTURE/playwright-argv"

  if [[ "$#" -ge 2 && "$1" == "run-code" && "$2" == --filename=* ]]; then
    local filename="${2#--filename=}"
    cp "$filename" "$WIKINO_BROWSE_TEST_CAPTURE/password-script"
    stat -c '%a' "$filename" > "$WIKINO_BROWSE_TEST_CAPTURE/password-mode"

    if [ "$WIKINO_BROWSE_TEST_PASSWORD_FAILS" = "1" ]; then
      printf '### Error
TimeoutError: locator.fill: Timeout 30000ms exceeded.
'
    fi
    printf '### Ran Playwright code
await ('
    cat "$filename"
    printf ')(page);
'
  fi

  return 0
}

sign_in_status() {
  if [[ "$WIKINO_BROWSE_TEST_TWO_FACTOR" == "1" && ! -f "$WIKINO_BROWSE_TEST_CAPTURE/status-called" ]]; then
    : > "$WIKINO_BROWSE_TEST_CAPTURE/status-called"
    printf 'NOT_SIGNED_IN https://example.test/sign_in/two_factor/new
'
    return
  fi

  printf 'SIGNED_IN https://example.test/
'
}

cmd_login "${1:-}"
`

// browseLoginPasswordは、シェル・JSON文字列・JavaScriptのソースがそれぞれ
// 特別扱いする文字を含む。受け渡しの各段がそれらに何をするのかを、仮定ではなく実際に
// 通すため。browseLoginEmailはdevcredsが1行目に返すアドレス。
const (
	browseLoginPassword = `p@ss word "$\quoted`
	browseLoginEmail    = "roster-user@example.com"
)

// browseLoginOptionsは、ハーネスの実行1回分の設定。2つのスイッチを位置では
// なく呼び出し側で名前付きにするのは、"false, true" がどちらの実行を求めているのかを
// 何も語らないため。
type browseLoginOptions struct {
	// requestedRoleはログインに指定する値。空文字列は何も指定しない実行を
	// 表し、既定を名指しする実行とは別物になる。既定を通すのは空のほうだけ。
	requestedRole string
	twoFactor     bool
	passwordFails bool
}

type browseLoginResult struct {
	stdout    string
	calls     string
	role      string
	totpEmail string
}

// TestBrowseLoginSelectsRoleは、既定アカウント、従来の番号指定2種、名簿の役割を
// そのまま指定する経路を確認する。各経路でログインが使う2行の資格情報解析も通す。
func TestBrowseLoginSelectsRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested string
		want      string
	}{
		{name: "指定が無いときはowner", requested: "", want: "owner"},
		{name: "1はowner", requested: "1", want: "owner"},
		{name: "2はcollaborator", requested: "2", want: "collaborator"},
		{name: "任意の役割はそのまま", requested: "guest", want: "guest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := runBrowseLogin(t, tt.requested, false)

			if result.role != tt.want {
				t.Errorf("資格情報を役割%qで検索することを期待したが%qだった", tt.want, result.role)
			}
			if !strings.Contains(result.stdout, tt.want+"としてログインした: https://example.test/") {
				t.Errorf("役割%qのログイン完了出力を期待したが%qだった", tt.want, result.stdout)
			}
		})
	}
}

// TestBrowseLoginPassesCredentialEmailToDevTOTPは2要素認証分岐を確認する。
// アドレスは名簿とずれうる別の環境変数ではなく、devcredsの1行目である必要がある。
func TestBrowseLoginPassesCredentialEmailToDevTOTP(t *testing.T) {
	t.Parallel()

	result := runBrowseLogin(t, "2", true)

	if result.role != "collaborator" {
		t.Errorf("2がcollaboratorとして検索されることを期待したが%qだった", result.role)
	}
	if result.totpEmail != "roster-user@example.com" {
		t.Errorf("devtotpへ名簿のメールアドレスを渡すことを期待したが%qだった", result.totpEmail)
	}
	if !strings.Contains(result.calls, "\tfill\tinput[name=\"totp_code\"]\t123456\t--submit\n") {
		t.Errorf("TOTPコードを送信する呼び出しを期待したが%qだった", result.calls)
	}
}

// TestBrowseLoginKeepsPasswordOutOfFailureOutputは、パスワード入力の手順が
// 失敗したときを扱う。playwright-cliはrun-codeへ、実行したコードを返す。ここでは
// それが生成スクリプトと、その中のパスワードになるため、応答の本文は、失敗時に
// 出してはならない唯一のものになる。
//
// 実行は失敗してそう告げ、Errorセクションは失敗を調査できるようにするため標準エラー
// 出力へ届き、応答のそれ以外は届かない。
func TestBrowseLoginKeepsPasswordOutOfFailureOutput(t *testing.T) {
	t.Parallel()

	captureDir := t.TempDir()

	command := browseLoginCommand(t, captureDir, browseLoginOptions{requestedRole: "owner", passwordFails: true})
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err == nil {
		t.Fatalf("パスワード入力に失敗したときは非ゼロ終了を期待したが成功した\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}

	if strings.Contains(stderr.String(), browseLoginPassword) {
		t.Errorf("標準エラー出力にパスワードが含まれている: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), browseLoginPassword) {
		t.Errorf("標準出力にパスワードが含まれている: %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "Ran Playwright code") {
		t.Errorf("実行したコードのセクションを出さないことを期待したが%qだった", stderr.String())
	}
	if !strings.Contains(stderr.String(), "TimeoutError: locator.fill") {
		t.Errorf("Errorセクションの内容を出すことを期待したが%qだった", stderr.String())
	}
	if !strings.Contains(stderr.String(), "パスワードの入力に失敗した") {
		t.Errorf("パスワード入力の失敗を告げる出力を期待したが%qだった", stderr.String())
	}

	// 明示的な削除は失敗した手順の後ろにあるため、抜ける途中でtrapが
	// スクリプトを持って行く必要がある。
	if _, err := os.Stat(filepath.Join(browseTmpDir(captureDir), "browse-cli.password.js")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("失敗時もパスワードスクリプトが削除されることを期待したがerr=%v", err)
	}
}

// browseLoginCommandはハーネスの実行を組み立てる。captureDirはスタブが記録
// するものと、browse.shを向けるtmpディレクトリの両方を持つため、テストは自身の
// t.TempDirの外に触れない。
func browseLoginCommand(t *testing.T, captureDir string, opts browseLoginOptions) *exec.Cmd {
	t.Helper()

	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "browse.sh"))
	if err != nil {
		t.Fatalf("browse.shの絶対パスを取得できなかった: %v", err)
	}

	args := []string{"-c", browseLoginHarness, "browse-test"}
	if opts.requestedRole != "" {
		args = append(args, opts.requestedRole)
	}

	command := exec.CommandContext(t.Context(), "bash", args...)
	command.Env = append(
		os.Environ(),
		"WIKINO_BROWSE_TEST_SCRIPT="+scriptPath,
		"WIKINO_BROWSE_TEST_CAPTURE="+captureDir,
		"WIKINO_BROWSE_TEST_PASSWORD="+browseLoginPassword,
		"WIKINO_BROWSE_TEST_EMAIL="+browseLoginEmail,
		"WIKINO_BROWSE_TEST_TWO_FACTOR="+boolFlag(opts.twoFactor),
		"WIKINO_BROWSE_TEST_PASSWORD_FAILS="+boolFlag(opts.passwordFails),
		"WIKINO_BROWSE_TMP_DIR="+browseTmpDir(captureDir),
	)

	return command
}

// browseTmpDirは、テスト中にbrowse.shが資格情報を含むファイルを書く場所。
// スクリプトから読み取るのではなくここで指定する。どこへ書くのかをスクリプトに
// 尋ねるテストは、誤った答えと正しい答えを区別できないため。
func browseTmpDir(captureDir string) string {
	return filepath.Join(captureDir, "tmp")
}

// boolFlagは、ハーネスが文字列比較で読むスイッチを組み立てる。
func boolFlag(on bool) string {
	if on {
		return "1"
	}

	return "0"
}

func runBrowseLogin(t *testing.T, requestedRole string, twoFactor bool) browseLoginResult {
	t.Helper()

	captureDir := t.TempDir()

	command := browseLoginCommand(t, captureDir, browseLoginOptions{requestedRole: requestedRole, twoFactor: twoFactor})
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("browse.shのログインテストに失敗: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("標準エラー出力が空であることを期待したが%qだった", stderr.String())
	}

	readCapture := func(name string) string {
		t.Helper()

		body, err := os.ReadFile(filepath.Join(captureDir, name))
		if err != nil {
			t.Fatalf("%sを読み込めなかった: %v", name, err)
		}

		return string(body)
	}

	calls := readCapture("playwright-argv")
	if strings.Contains(calls, browseLoginPassword) {
		t.Errorf("playwright-cliのargvにパスワードが含まれている: %q", calls)
	}
	if !strings.Contains(calls, "\tfill\tinput[name=\"email\"]\t"+browseLoginEmail+"\n") {
		t.Errorf("devcredsの1行目をemail入力に使うことを期待したが%qだった", calls)
	}

	passwordScript := readCapture("password-script")
	quotedPassword, err := json.Marshal(browseLoginPassword)
	if err != nil {
		t.Fatalf("パスワードをJSON文字列へ変換できなかった: %v", err)
	}
	if !strings.Contains(passwordScript, string(quotedPassword)) {
		t.Errorf("devcredsの2行目をパスワード入力に使うことを期待したが%qだった", passwordScript)
	}
	if mode := strings.TrimSpace(readCapture("password-mode")); mode != "600" {
		t.Errorf("パスワードスクリプトの権限が600であることを期待したが%qだった", mode)
	}
	// playwright-cliへ渡されたパスは、このテストがbrowse.shを向けた先で
	// ある必要がある。これが無いと、下の削除の確認は、まったく別の場所へスクリプトを
	// 書いてそのまま残した実行に対しても成立してしまう。
	if want := "\t--filename=" + filepath.Join(browseTmpDir(captureDir), "browse-cli.password.js") + "\n"; !strings.Contains(calls, want) {
		t.Errorf("パスワードスクリプトを%qへ書くことを期待したが%qだった", want, calls)
	}
	if _, err := os.Stat(filepath.Join(browseTmpDir(captureDir), "browse-cli.password.js")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ログイン後にパスワードスクリプトが削除されることを期待したがerr=%v", err)
	}

	var totpEmail string
	totpEmailBody, err := os.ReadFile(filepath.Join(captureDir, "devtotp-email"))
	switch {
	case err == nil:
		totpEmail = string(totpEmailBody)
	case errors.Is(err, os.ErrNotExist):
	default:
		t.Fatalf("devtotpへ渡したメールアドレスを読み込めなかった: %v", err)
	}

	return browseLoginResult{
		stdout:    stdout.String(),
		calls:     calls,
		role:      readCapture("role"),
		totpEmail: totpEmail,
	}
}
