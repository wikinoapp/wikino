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

// browseLoginHarness sources browse.sh for its functions and replaces what
// reaches outside the process: the Go toolchain, playwright-cli, and the
// Basic-auth config that would need a base URL to build. Everything the test
// is about — role translation, the two-line credential parsing, the generated
// password script, the two-factor branch — runs as written.
//
// The file paths are not restated here. browse.sh takes them from
// WIKINO_BROWSE_TMP_DIR, so its own definitions stay the only ones, and a
// rename there cannot leave this harness pointing somewhere else while the
// script writes a real password file to the developer's tmp directory.
//
// [Ja] browseLoginHarness は browse.sh を source して関数を取り込み、プロセスの外へ
// 出るもの (Go ツールチェイン、playwright-cli、ベース URL が要る Basic 認証 config の
// 生成) だけを差し替える。テストの対象そのもの (役割の読み替え、2 行の資格情報解析、
// 生成されるパスワードスクリプト、2 要素認証分岐) は書かれたまま動く。
//
// ファイルパスはここに書き写さない。browse.sh が WIKINO_BROWSE_TMP_DIR から取るため、
// 定義は browse.sh のものだけになり、あちらで改名したときに、このハーネスだけが別の
// 場所を指したまま、スクリプトが開発者の実 tmp ディレクトリへパスワードのファイルを
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

# pw stands in for playwright-cli itself rather than for pw_checked, so that
# both wrappers run as written and the argv captured here is the argv the real
# command would have been given.
#
# run-code is answered the way playwright-cli answers it: with the code it ran
# echoed back, which for the sign-in is the generated script and the password in
# it. That is the response pw_checked_secret exists to keep off the terminal.
# The js code fence the real response puts around it is left out, because a Go
# raw string literal cannot carry a backtick; the filter keys on the "### "
# section headers, which are reproduced.
#
# [Ja] pw は pw_checked ではなく playwright-cli 自体の代役。どちらのラッパーも書かれた
# まま動かし、ここで捕捉する argv を、実際のコマンドが渡されたはずの argv にするため。
#
# run-code には playwright-cli と同じ形で答える。実行したコードを返す形であり、
# サインインではそれが生成スクリプトと、その中のパスワードになる。これが
# pw_checked_secret の存在理由である応答そのものになる。実際の応答がその周りに置く
# js のコードフェンスは省いた。Go の raw string literal はバッククォートを持てないため。
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

// browseLoginPassword carries the characters a shell, a JSON string and a
// JavaScript source each treat specially, so that what every step of the
// hand-off does to them is exercised rather than assumed. browseLoginEmail is
// the address devcreds returns on the first line.
//
// [Ja] browseLoginPassword は、シェル・JSON 文字列・JavaScript のソースがそれぞれ
// 特別扱いする文字を含む。受け渡しの各段がそれらに何をするのかを、仮定ではなく実際に
// 通すため。browseLoginEmail は devcreds が 1 行目に返すアドレス。
const (
	browseLoginPassword = `p@ss word "$\quoted`
	browseLoginEmail    = "roster-user@example.com"
)

// browseLoginOptions is what one harness run is set up with. The two switches
// are named at the call site rather than passed as bare positions, because
// "false, true" says nothing about which run is being asked for.
//
// [Ja] browseLoginOptions は、ハーネスの実行 1 回分の設定。2 つのスイッチを位置では
// なく呼び出し側で名前付きにするのは、"false, true" がどちらの実行を求めているのかを
// 何も語らないため。
type browseLoginOptions struct {
	// requestedRole is what the login is asked for. Empty stands for an
	// invocation that names nothing, which is not the same as naming the
	// default: only the empty one exercises the default.
	//
	// [Ja] requestedRole はログインに指定する値。空文字列は何も指定しない実行を
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

// TestBrowseLoginSelectsRole covers the default account, both legacy numeric
// aliases, and a roster role passed through unchanged. Each path also exercises
// the two-line credential parsing used by login.
//
// [Ja] TestBrowseLoginSelectsRole は、既定アカウント、従来の番号指定 2 種、名簿の役割を
// そのまま指定する経路を確認する。各経路でログインが使う 2 行の資格情報解析も通す。
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
				t.Errorf("資格情報を役割 %q で検索することを期待したが %q だった", tt.want, result.role)
			}
			if !strings.Contains(result.stdout, "logged in as "+tt.want+": https://example.test/") {
				t.Errorf("役割 %q のログイン完了出力を期待したが %q だった", tt.want, result.stdout)
			}
		})
	}
}

// TestBrowseLoginPassesCredentialEmailToDevTOTP covers the two-factor branch.
// The address has to be the first line returned by devcreds, rather than a
// separate environment variable that can drift from the roster.
//
// [Ja] TestBrowseLoginPassesCredentialEmailToDevTOTP は 2 要素認証分岐を確認する。
// アドレスは名簿とずれうる別の環境変数ではなく、devcreds の 1 行目である必要がある。
func TestBrowseLoginPassesCredentialEmailToDevTOTP(t *testing.T) {
	t.Parallel()

	result := runBrowseLogin(t, "2", true)

	if result.role != "collaborator" {
		t.Errorf("2がcollaboratorとして検索されることを期待したが %q だった", result.role)
	}
	if result.totpEmail != "roster-user@example.com" {
		t.Errorf("devtotpへ名簿のメールアドレスを渡すことを期待したが %q だった", result.totpEmail)
	}
	if !strings.Contains(result.calls, "\tfill\tinput[name=\"totp_code\"]\t123456\t--submit\n") {
		t.Errorf("TOTPコードを送信する呼び出しを期待したが %q だった", result.calls)
	}
}

// TestBrowseLoginKeepsPasswordOutOfFailureOutput covers what happens when the
// password step fails. playwright-cli answers run-code with the code it ran,
// which here is the generated script and the password in it, so the response
// body is the one thing the failure must not print.
//
// The run fails and says so, the Error section reaches stderr because that is
// what makes the failure diagnosable, and nothing else from the response does.
//
// [Ja] TestBrowseLoginKeepsPasswordOutOfFailureOutput は、パスワード入力の手順が
// 失敗したときを扱う。playwright-cli は run-code へ、実行したコードを返す。ここでは
// それが生成スクリプトと、その中のパスワードになるため、応答の本文は、失敗時に
// 出してはならない唯一のものになる。
//
// 実行は失敗してそう告げ、Error セクションは失敗を調査できるようにするため標準エラー
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
		t.Errorf("実行したコードのセクションを出さないことを期待したが %q だった", stderr.String())
	}
	if !strings.Contains(stderr.String(), "TimeoutError: locator.fill") {
		t.Errorf("Errorセクションの内容を出すことを期待したが %q だった", stderr.String())
	}
	if !strings.Contains(stderr.String(), "filling the password failed") {
		t.Errorf("パスワード入力の失敗を告げる出力を期待したが %q だった", stderr.String())
	}

	// The trap has to take the script with it on the way out, since the
	// explicit removal sits after the step that failed.
	//
	// [Ja] 明示的な削除は失敗した手順の後ろにあるため、抜ける途中で trap が
	// スクリプトを持って行く必要がある。
	if _, err := os.Stat(filepath.Join(browseTmpDir(captureDir), "browse-cli.password.js")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("失敗時もパスワードスクリプトが削除されることを期待したが err=%v", err)
	}
}

// browseLoginCommand builds the harness invocation. captureDir holds both what
// the stubs record and the tmp directory browse.sh is pointed at, so a test
// touches nothing outside its own t.TempDir.
//
// [Ja] browseLoginCommand はハーネスの実行を組み立てる。captureDir はスタブが記録
// するものと、browse.sh を向ける tmp ディレクトリの両方を持つため、テストは自身の
// t.TempDir の外に触れない。
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

// browseTmpDir is where browse.sh writes its credential-bearing files during a
// test. It is named here rather than read back from the script, because a test
// that asked the script where it writes could not tell a wrong answer from a
// right one.
//
// [Ja] browseTmpDir は、テスト中に browse.sh が資格情報を含むファイルを書く場所。
// スクリプトから読み取るのではなくここで指定する。どこへ書くのかをスクリプトに
// 尋ねるテストは、誤った答えと正しい答えを区別できないため。
func browseTmpDir(captureDir string) string {
	return filepath.Join(captureDir, "tmp")
}

// boolFlag renders a switch the harness reads with a string comparison.
//
// [Ja] boolFlag は、ハーネスが文字列比較で読むスイッチを組み立てる。
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
		t.Errorf("標準エラー出力が空であることを期待したが %q だった", stderr.String())
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
		t.Errorf("devcredsの1行目をemail入力に使うことを期待したが %q だった", calls)
	}

	passwordScript := readCapture("password-script")
	quotedPassword, err := json.Marshal(browseLoginPassword)
	if err != nil {
		t.Fatalf("パスワードをJSON文字列へ変換できなかった: %v", err)
	}
	if !strings.Contains(passwordScript, string(quotedPassword)) {
		t.Errorf("devcredsの2行目をパスワード入力に使うことを期待したが %q だった", passwordScript)
	}
	if mode := strings.TrimSpace(readCapture("password-mode")); mode != "600" {
		t.Errorf("パスワードスクリプトの権限が600であることを期待したが %q だった", mode)
	}
	// The path playwright-cli was given has to be the one this test pointed
	// browse.sh at. Without this, the removal check below would hold for a run
	// that wrote the script somewhere else entirely and left it there.
	//
	// [Ja] playwright-cli へ渡されたパスは、このテストが browse.sh を向けた先で
	// ある必要がある。これが無いと、下の削除の確認は、まったく別の場所へスクリプトを
	// 書いてそのまま残した実行に対しても成立してしまう。
	if want := "\t--filename=" + filepath.Join(browseTmpDir(captureDir), "browse-cli.password.js") + "\n"; !strings.Contains(calls, want) {
		t.Errorf("パスワードスクリプトを %q へ書くことを期待したが %q だった", want, calls)
	}
	if _, err := os.Stat(filepath.Join(browseTmpDir(captureDir), "browse-cli.password.js")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ログイン後にパスワードスクリプトが削除されることを期待したが err=%v", err)
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
