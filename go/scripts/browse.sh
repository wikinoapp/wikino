#!/usr/bin/env bash
#
# browse.sh drives playwright-cli for browser verification of the dev site:
# it generates the Basic-auth config, runs the dev sign-in (clearing the
# two-factor step when the account has it enabled), reuses the logged-in
# session for screenshots, and cleans up.
#
# The sign-in credentials come from the roster the seed creates its accounts
# from (go/seed-users.toml), read through `wikino devcreds`, so that the account
# signed in as here is the account the seed actually created. The base URL is
# still expected in KORYLUS_BROWSING_BASE_URL, so run this under the op-run
# wrapper (see the browse-* targets in go/Makefile). Reading that URL through
# op-run avoids evaluating the .env in a shell, which would corrupt the
# basic-auth credentials it carries when they contain a `$`. The dev server must
# run with Turnstile disabled (WIKINO_TURNSTILE_ENABLED=false in the dev .env)
# or bot verification blocks the sign-in submit. Accounts with two-factor auth
# enabled additionally need the dev database to be reachable, since the TOTP
# code is generated from user_two_factor_auths.
#
# [Ja] browse.sh は playwright-cli を駆動して dev サイトのブラウザ確認を行う。
# Basic 認証 config の生成・dev サインイン (アカウントが 2 要素認証を有効にしている
# 場合はそのステップの通過も含む)・ログイン済みセッションでのスクショ・後片付けを
# まとめる。
#
# サインインに使う資格情報は、シードがアカウントを作成する元にしている名簿
# (go/seed-users.toml) から `wikino devcreds` を通して読む。ここでサインインする
# アカウントを、シードが実際に作成したアカウントそのものにするため。ベース URL は
# 引き続き KORYLUS_BROWSING_BASE_URL を前提とするため、op run ラッパー配下
# (go/Makefile の browse-* ターゲット) から実行する。この URL を op run 経由で読む
# ことで、.env をシェル評価して、そこに含まれる Basic 認証 creds を壊すのを避ける
# (creds が `$` を含む場合)。dev サーバは Turnstile を
# 無効化 (dev の .env で WIKINO_TURNSTILE_ENABLED=false) して起動している必要が
# あり、でないと Bot 検証でサインインの送信が弾かれる。2 要素認証を有効にした
# アカウントを使う場合は、TOTP コードを user_two_factor_auths から生成するため
# dev DB へ到達できることも前提になる。
set -euo pipefail

SESSION=dev
# The account signed in as when none is named. The owner administers both spaces
# and holds every feature flag, so it is the account that reaches the most
# screens.
#
# [Ja] 役割の指定が無いときにサインインするアカウント。owner は両スペースを管理し、
# フィーチャーフラグを全件持つため、最も多くの画面へ到達できる。
DEFAULT_ROLE=owner
# The Go module root, derived from this script's location so the helpers this
# script runs — wikino devcreds and devtotp — resolve no matter which directory
# it is invoked from. It is also where the roster is looked for, which
# wikino devcreds names relative to the module root.
#
# [Ja] Go モジュールのルート。本スクリプトの位置から求めることで、どのディレクトリ
# から実行してもヘルパー (wikino devcreds と devtotp) を解決できるようにする。
# 名簿を探す基準もここになる。wikino devcreds が名簿のパスをモジュールルートからの
# 相対で持っているため。
GO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# Where the credential-bearing files and the screenshots are written. It is the
# repository's gitignored tmp directory, and WIKINO_BROWSE_TMP_DIR points it
# elsewhere for the subprocess tests. Giving the tests one seam is what keeps
# the file names below the only definition of themselves: a test that copied
# them would go on passing after a rename here while this script wrote the real
# password file to the developer's tmp directory.
#
# [Ja] 資格情報を含むファイルとスクショの書き込み先。リポジトリの gitignore 済み
# tmp ディレクトリであり、サブプロセステストは WIKINO_BROWSE_TMP_DIR で別の場所へ
# 向ける。テストへ差し込み口を 1 つだけ渡すことが、以下のファイル名の定義をここ
# だけにする方法になる。テスト側で名前を書き写すと、ここで改名したときにテストは
# 通ったまま、本スクリプトが開発者の実 tmp ディレクトリへパスワードのファイルを
# 書くことになるため。
TMP_DIR="${WIKINO_BROWSE_TMP_DIR:-/workspace/tmp}"
CONFIG_FILE="$TMP_DIR/browse-cli.config.json"
PASSWORD_SCRIPT_FILE="$TMP_DIR/browse-cli.password.js"
ORIGIN_FILE="$TMP_DIR/browse-cli.origin"
SHOT_DIR="$TMP_DIR/browse"
LEGACY_PROFILE_DIR="$TMP_DIR/browse-cli-profile"

# playwright-cli's Chromium is baked into a dedicated browsers path
# (WIKINO_PLAYWRIGHT_CLI_BROWSERS_PATH), separate from the E2E Chromium under the
# container's default PLAYWRIGHT_BROWSERS_PATH (/ms-playwright). Point Playwright
# at the playwright-cli path so it resolves the matching Chromium build instead
# of the E2E one.
#
# [Ja] playwright-cli の Chromium は E2E とは別の専用パス
# (WIKINO_PLAYWRIGHT_CLI_BROWSERS_PATH) に焼かれている。コンテナ既定の
# PLAYWRIGHT_BROWSERS_PATH (/ms-playwright) は E2E 用の Chromium を指すため、
# playwright-cli 用のパスに向けて、E2E ではなく対応する Chromium ビルドを
# 解決させる。
export PLAYWRIGHT_BROWSERS_PATH="${WIKINO_PLAYWRIGHT_CLI_BROWSERS_PATH:-/ms-playwright-cli}"

pw() { playwright-cli -s="$SESSION" "$@"; }

# pw_checked handles command-level Playwright errors, which playwright-cli
# reports in its output while still exiting with status 0. Callers may discard
# successful output, but errors are always preserved on stderr.
#
# [Ja] pw_checked は、playwright-cli が終了コード 0 のまま出力へ記録する
# Playwright のコマンドレベルエラーを検出する。呼び出し側は成功時の出力を
# 捨てられるが、エラーは常に stderr へ残す。
pw_checked() {
  local output
  if ! output="$(pw "$@" 2>&1)"; then
    printf '%s\n' "$output" >&2
    return 1
  fi
  if [[ "$output" == *"### Error"* ]]; then
    printf '%s\n' "$output" >&2
    return 1
  fi
  printf '%s\n' "$output"
}

# pw_checked_secret is pw_checked for a command whose code carries a secret. It
# never lets the response body reach the terminal.
#
# run-code echoes the code it ran back to the caller in a
# "### Ran Playwright code" section, and for the sign-in that code is the
# generated script, password and all. Nothing leaks today — the successful
# response is discarded and playwright-cli answers a thrown error with the
# Error section alone — but both of those are its choices to change. Relaying
# only the Error section is what keeps the password off the terminal, the CI log
# and the agent transcript when a response gains a section: a section this
# function was not told to relay is not relayed.
#
# Output carrying no section header at all is not a response. It is
# playwright-cli failing before it read anything, so it is relayed whole, which
# is what leaves a missing binary or a dead session diagnosable.
#
# [Ja] pw_checked_secret は、コードが秘密を含むコマンド用の pw_checked。応答の本文を
# 端末へ出さない。
#
# run-code は実行したコードを "### Ran Playwright code" セクションとして呼び出し側へ
# 返すが、サインインではそのコードが、パスワードを含む生成スクリプトそのものになる。
# 現状は漏れない (成功時の応答は捨てており、playwright-cli は throw されたエラーへ
# Error セクションだけを返す) が、どちらも playwright-cli 側が変えられるものである。
# Error セクションだけを流すことが、応答にセクションが増えたときにも、端末・CI ログ・
# エージェントのトランスクリプトへパスワードを出さない方法になる。この関数が流すと
# 指示されていないセクションは流れない。
#
# セクションの見出しを 1 つも持たない出力は応答ではない。playwright-cli が何かを読む
# 前に失敗した場合であるため、そのまま流す。バイナリが無い場合やセッションが死んで
# いる場合を、調査できる形で残すため。
pw_checked_secret() {
  local output
  local status=0
  output="$(pw "$@" 2>&1)" || status=$?
  if [ "$status" -eq 0 ] && [[ "$output" != *"### Error"* ]]; then
    return 0
  fi

  {
    if [[ "$output" == *"### "* ]]; then
      printf '%s\n' "$output" | awk '/^### /{ relay = ($0 == "### Error") } relay'
    else
      printf '%s\n' "$output"
    fi
  } >&2

  return 1
}

# sign_in_status prints "<SIGNED_IN|NOT_SIGNED_IN> <url>" for the page the
# browser currently sits on. A completed sign-in redirects away from /sign_in
# (to home or the back URL); staying under /sign_in means not signed in — the
# form was re-rendered (422), or the account stopped at the two-factor
# challenge. The pathname is extracted with a regex rather than `new URL()`:
# playwright-cli's run-code runs in a sandbox where the URL constructor is not
# defined. The decision is returned as a sentinel and acted on in bash so a
# failed sign-in can exit non-zero (a throw inside run-code only prints an error
# and exits 0).
#
# [Ja] sign_in_status は、ブラウザが現在いるページについて
# "<SIGNED_IN|NOT_SIGNED_IN> <url>" を出力する。サインインが完了すると /sign_in から
# 離れる (ホームか back URL へ)。/sign_in 配下に留まる場合は未ログインを意味し、
# フォームの再描画 (422) か、2 要素認証のチャレンジで止まっているかのいずれか。
# pathname は `new URL()` ではなく正規表現で取り出す。playwright-cli の run-code は
# URL コンストラクタが未定義のサンドボックスで動くため。判定は sentinel で返して
# bash 側で扱い、失敗時に非ゼロ終了できるようにする (run-code 内の throw はエラーを
# 表示するだけで終了コードは 0 になるため)。
sign_in_status() {
  local result
  result="$(pw_checked --raw run-code "async page => {
    await page.waitForLoadState('networkidle');
    const href = page.url();
    const path = href.replace(/^[a-z][a-z0-9+.-]*:\/\/[^/]+/i, '').replace(/[?#].*/, '');
    const notSignedIn = path === '/sign_in' || path.startsWith('/sign_in/');
    return (notSignedIn ? 'NOT_SIGNED_IN ' : 'SIGNED_IN ') + href;
  }")"

  # --raw wraps the returned string in double quotes; strip them before matching.
  #
  # [Ja] --raw は返り値の文字列を二重引用符で囲むため、判定前に取り除く。
  result="${result%\"}"
  result="${result#\"}"
  printf '%s\n' "$result"
}

# build_config writes the Basic-auth config (httpCredentials) parsed from
# KORYLUS_BROWSING_BASE_URL, plus a credential-free origin file for later
# navigation. The config carries the credentials, so it is written 0600 under
# the gitignored tmp dir and removed as soon as login captures it in the
# browser context.
#
# [Ja] build_config は KORYLUS_BROWSING_BASE_URL から Basic 認証 config
# (httpCredentials) を生成し、以降の遷移用に creds を抜いた origin ファイルも書く。
# config は creds を含むため gitignore 済み tmp に 0600 で書き、ログインが
# ブラウザコンテキストに取り込んだ直後に削除する。
build_config() {
  mkdir -p "$TMP_DIR"
  node -e '
    const fs = require("fs");
    const raw = process.env.KORYLUS_BROWSING_BASE_URL || "";
    if (!raw) { console.error("KORYLUS_BROWSING_BASE_URL is not set"); process.exit(1); }
    const u = new URL(raw);
    const cfg = { browser: { contextOptions: { httpCredentials: {
      username: decodeURIComponent(u.username),
      password: decodeURIComponent(u.password),
    } } } };
    fs.writeFileSync(process.argv[1], JSON.stringify(cfg), { mode: 0o600 });
    u.username = u.password = "";
    fs.writeFileSync(process.argv[2], u.origin);
  ' "$CONFIG_FILE" "$ORIGIN_FILE"
}

# build_password_script writes the password input and submit actions to a 0600
# file. The password reaches Node over stdin and playwright-cli receives only
# the file path, keeping the secret out of both processes' argv.
#
# [Ja] build_password_script は、パスワード入力と送信の操作を 0600 のファイルへ
# 書く。パスワードは標準入力で Node へ渡し、playwright-cli にはファイルパスだけを
# 渡すことで、どちらのプロセスの argv にも秘密を載せない。
build_password_script() {
  node -e '
    const fs = require("fs");
    const password = fs.readFileSync(0, "utf8");
    const selector = "input[name=\"password\"]";
    const source = `async page => {
      const input = page.locator(${JSON.stringify(selector)});
      await input.fill(${JSON.stringify(password)});
      await input.press("Enter");
    }`;
    fs.rmSync(process.argv[1], { force: true });
    fs.writeFileSync(process.argv[1], source, { mode: 0o600 });
  ' "$PASSWORD_SCRIPT_FILE"
}

# role_for translates what was asked for into a role the roster holds. The
# roster names accounts by role; before it existed this script took the index of
# a KORYLUS_BROWSING_USER{N} pair, so the two indexes that ever existed are
# accepted as the roles they stood for. The translation lives here rather than
# on the Go side so that the roster never learns the numbering, and dropping the
# compatibility is a change to this function alone.
#
# [Ja] role_for は、指定された値を名簿が持つ役割へ読み替える。名簿はアカウントを
# 役割で名指しするが、名簿ができる前の本スクリプトは KORYLUS_BROWSING_USER{N} の組の
# 番号を受け取っていたため、当時存在した 2 つの番号を、それが指していた役割として
# 受け付ける。読み替えを Go 側ではなくここに置いているのは、名簿にこの番号を
# 知らせないためで、互換をやめるときの変更もこの関数だけで済む。
role_for() {
  case "$1" in
    1) printf 'owner\n' ;;
    2) printf 'collaborator\n' ;;
    *) printf '%s\n' "$1" ;;
  esac
}

# CURL_CONFIG_FILE carries the Basic-auth credentials for the reachability check. curl takes them
# from a 0600 config file rather than from -u, so that they never appear in the process arguments
# the way the password script keeps them off argv for playwright-cli.
#
# [Ja] CURL_CONFIG_FILE は到達確認で使う Basic 認証の資格情報を持つ。curl には -u ではなく 0600 の
# 設定ファイルから読ませ、パスワードスクリプトが playwright-cli に対してそうしているのと同じく、
# プロセスの引数に資格情報が現れないようにする。
CURL_CONFIG_FILE="$TMP_DIR/browse-cli.curlrc"

# check_environment validates what browser verification needs, without printing any credential
# value. Keeping the check in browse.sh makes diagnostics and login go through the same op-run
# wrapper and the same requirements: a diagnostic assembled by hand can drop one of them and report
# a healthy environment as broken.
#
# The last step asks the dev URL for a page and looks at who answered. The reverse proxy hands
# every path the Go version does not claim to the Rails version, so an unreachable Go app does not
# fail loudly — it answers with whatever Rails makes of the same path, which for a Go-only screen
# is a 404. Checking for the CSRF cookie the Go version always sets is what tells the two apart.
#
# [Ja] check_environment は、ブラウザ確認に必要なものを、資格情報の値を出さずに検証する。検査を
# browse.sh に置くことで、診断とログインが同じ op run ラッパーと同じ要件を通る。手で組み立てた
# 診断は要件を 1 つ落とすことがあり、健全な環境を壊れていると報告してしまう。
#
# 最後の手順では dev URL からページを取得し、誰が応答したのかを見る。リバースプロキシは Go 版が
# 引き受けないパスをすべて Rails 版へ渡すため、Go 版へ到達できない状態は大きな音を立てて失敗せず、
# 同じパスを Rails 版が解釈した結果 (Go 版にしかない画面なら 404) が返ってくる。Go 版が必ず設定する
# CSRF クッキーの有無が、この 2 つを見分ける手がかりになる。
check_environment() {
  local failed=0

  if [ "${APP_ENV:-}" != "dev" ]; then
    echo "APP_ENV must be dev (run through the Makefile browse target)" >&2
    failed=1
  fi

  if [ -z "${KORYLUS_BROWSING_BASE_URL:-}" ]; then
    echo "KORYLUS_BROWSING_BASE_URL is not set" >&2
    failed=1
  fi

  if [ "${WIKINO_TURNSTILE_ENABLED:-}" != "false" ]; then
    echo "WIKINO_TURNSTILE_ENABLED must be false for dev browser login" >&2
    failed=1
  fi

  local command_name
  for command_name in node curl playwright-cli; do
    if ! command -v "$command_name" >/dev/null 2>&1; then
      echo "$command_name is not available" >&2
      failed=1
    fi
  done

  if [ -n "${KORYLUS_BROWSING_BASE_URL:-}" ] && command -v node >/dev/null 2>&1; then
    if ! node -e '
      const raw = process.env.KORYLUS_BROWSING_BASE_URL || "";
      let url;
      try {
        url = new URL(raw);
      } catch {
        console.error("KORYLUS_BROWSING_BASE_URL must be a valid URL");
        process.exit(1);
      }
      if (url.protocol !== "http:" && url.protocol !== "https:") {
        console.error("KORYLUS_BROWSING_BASE_URL must use http or https");
        process.exit(1);
      }
      if (!url.username || !url.password) {
        console.error("KORYLUS_BROWSING_BASE_URL must include Basic-auth credentials");
        process.exit(1);
      }
    '; then
      failed=1
    fi
  fi

  if [ "$failed" -ne 0 ]; then
    return 1
  fi
}


# check_credentials reports whether the roster holds the role being asked for. It is part of the
# diagnosis rather than of check_environment, because the login path reads the same credentials
# straight afterwards and reports the same failure itself: running it here as well would ask the
# roster twice on every login.
#
# [Ja] check_credentials は、尋ねている役割を名簿が持っているかを報告する。check_environment では
# なく診断の側に置いているのは、ログインの経路がこの直後に同じ資格情報を読み、同じ失敗を自ら
# 報告するためである。ここでも実行すると、ログインのたびに名簿へ 2 度尋ねることになる。
check_credentials() {
  local role
  role="$(role_for "${1:-$DEFAULT_ROLE}")"

  if ! (cd "$GO_DIR" && go run ./cmd/wikino devcreds "$role" >/dev/null 2>&1); then
    echo "no credentials for role '$role': check the roster wikino devcreds reads" >&2

    return 1
  fi
}

# check_reachable fetches the sign-in screen through the dev URL and reports which application
# answered. The credentials reach curl through a 0600 config file, and the file is removed as soon
# as the request is done.
#
# [Ja] check_reachable は dev URL からサインイン画面を取得し、どのアプリケーションが応答したかを
# 報告する。資格情報は 0600 の設定ファイル経由で curl へ渡し、リクエストが終わり次第そのファイルを
# 削除する。
check_reachable() (
  # Keep the cleanup trap scoped to this check. It runs for normal returns, command failures, and
  # signals without replacing the login command's separate credential-file trap.
  #
  # [Ja] 後片付けの trap はこの検査の subshell 内だけに置く。ログインコマンド側にある別の
  # 資格情報ファイル用 trap を置き換えず、正常 return・コマンド失敗・signal のすべてで実行する。
  trap 'rm -f "$CURL_CONFIG_FILE"' EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM

  mkdir -p "$TMP_DIR"
  node -e '
    const fs = require("fs");
    const u = new URL(process.env.KORYLUS_BROWSING_BASE_URL);
    const user = decodeURIComponent(u.username);
    const pass = decodeURIComponent(u.password);
    if (/[\u0000-\u001f\u007f-\u009f]/u.test(user) || /[\u0000-\u001f\u007f-\u009f]/u.test(pass)) {
      console.error("KORYLUS_BROWSING_BASE_URL credentials must not include control characters");
      process.exit(1);
    }
    const escapeCurlConfigString = (value) => value.replace(/\\/g, "\\\\").replace(/"/g, "\\\"");
    fs.rmSync(process.argv[1], { force: true });
    fs.writeFileSync(
      process.argv[1],
      `user = "${escapeCurlConfigString(`${user}:${pass}`)}"\n`,
      { mode: 0o600 },
    );
    u.username = u.password = "";
    fs.writeFileSync(process.argv[2], u.origin);
  ' "$CURL_CONFIG_FILE" "$ORIGIN_FILE"

  local origin
  origin="$(cat "$ORIGIN_FILE")"
  local response
  local curl_status=0
  response="$(curl -s -o /dev/null -K "$CURL_CONFIG_FILE" -w '%{http_code} %{header_json}' "$origin/sign_in")" || curl_status=$?
  rm -f "$CURL_CONFIG_FILE"

  if [ "$curl_status" -ne 0 ]; then
    echo "dev URL did not respond" >&2

    return 1
  fi

  local status="${response%% *}"
  case "$status" in
    200) ;;
    401)
      echo "dev URL rejected the Basic-auth credentials in KORYLUS_BROWSING_BASE_URL" >&2

      return 1
      ;;
    *)
      echo "dev URL answered $status for /sign_in" >&2

      return 1
      ;;
  esac

  # The Go version sets its CSRF cookie on every response, so its absence means the request reached
  # the Rails version instead — the sign-in screen is served by Go.
  #
  # [Ja] Go 版はすべての応答に CSRF クッキーを設定するため、それが無いということは、リクエストが
  # Rails 版へ届いたことを意味する。サインイン画面を提供するのは Go 版である。
  if ! printf '%s' "$response" | grep -q "wikino_csrf_token"; then
    echo "dev URL answered without the Go version's CSRF cookie: the request is not reaching the Go app" >&2

    return 1
  fi
)

# cmd_check reports whether the environment browser verification needs is in place, so that a
# failure is read where it is caused instead of being chased through a hand-written diagnostic.
#
# [Ja] cmd_check は、ブラウザ確認に必要な環境が整っているかを報告する。失敗を、手書きの診断で
# 追いかけるのではなく、原因のある場所で読めるようにするため。
cmd_check() {
  check_environment
  check_credentials "${1:-}"
  check_reachable
  echo "browser environment ready for $(role_for "${1:-$DEFAULT_ROLE}")"
}

cmd_login() {
  local role
  role="$(role_for "${1:-$DEFAULT_ROLE}")"

  # The credentials come from the roster the seed reads (go/seed-users.toml),
  # by way of `wikino devcreds`, which prints the email and the password one per
  # line. Asking the file the seed asked is what keeps the sign-in that follows
  # a seed working: an account cannot be changed in one place and left as it was
  # in the other.
  #
  # The password stays off argv throughout: it arrives on stdout, crosses into
  # Node on stdin, and is written to a 0600 script. playwright-cli receives only
  # that script's path, which is removed immediately after the input action, and
  # the response that carries the script back is handled by pw_checked_secret.
  #
  # [Ja] 資格情報は、シードが読むのと同じ名簿 (go/seed-users.toml) から
  # `wikino devcreds` を通して受け取る。devcreds はメールアドレスとパスワードを
  # 1 行ずつ出力する。シードが尋ねたファイルに尋ねることが、シード直後のサインインが
  # 通り続ける理由になる。片方だけを変えてもう片方が元のまま、ということが起こらない
  # ため。パスワードは全経路で argv に載せず、標準出力から Node の標準入力を経て
  # 0600 のスクリプトへ書く。playwright-cli にはそのパスだけを渡して入力直後に削除し、
  # そのスクリプトを返してくる応答は pw_checked_secret が扱う。
  local credentials
  credentials="$(cd "$GO_DIR" && go run ./cmd/wikino devcreds "$role")"
  local lines=()
  mapfile -t lines <<<"$credentials"
  local email="${lines[0]:-}"
  local pass="${lines[1]:-}"
  if [ "${#lines[@]}" -ne 2 ] || [ -z "$email" ] || [ -z "$pass" ]; then
    echo "no credentials for role '$role': wikino devcreds must print exactly two non-empty lines" >&2
    exit 1
  fi

  # Remove credential-bearing files on any exit, so a mid-login failure never
  # leaves the Basic-auth config or password script at rest.
  #
  # [Ja] 資格情報を含むファイルをどの終了経路でも削除し、ログイン途中の失敗でも
  # Basic 認証 config やパスワードスクリプトをディスクに残さない。
  trap 'rm -f "$CONFIG_FILE" "$PASSWORD_SCRIPT_FILE"' EXIT

  build_config
  local origin
  origin="$(cat "$ORIGIN_FILE")"

  # Basic auth is passed via the config (httpCredentials). The browser is opened
  # without a persistent profile: playwright-cli keeps the named session's
  # browser alive between shell invocations, so the login cookies survive in the
  # context anyway, and a persistent profile stops delivering input events to
  # the page after the first navigation (the two-factor form can be filled but
  # never submitted).
  #
  # [Ja] Basic 認証は config (httpCredentials) で渡す。ブラウザは永続プロファイル
  # 無しで開く。playwright-cli は名前付きセッションのブラウザをシェル呼び出しの間も
  # 生かし続けるため、ログイン Cookie はコンテキスト内で残る。加えて永続プロファイル
  # では最初の遷移以降ページへ入力イベントが届かなくなる (2 要素認証フォームに入力は
  # できても送信できない)。
  pw_checked open "$origin/sign_in" --browser=chromium --config="$CONFIG_FILE" >/dev/null

  # Wikino's sign-in is a single-step form (email + password submitted together).
  # Fill email without submitting, then submit with Enter on the password field.
  # Turnstile must be disabled (WIKINO_TURNSTILE_ENABLED=false) or the submit is
  # blocked.
  #
  # [Ja] Wikino のサインインは単一ステップのフォーム (email + password を一括送信)。
  # email は送信せずに入力し、password で Enter を押して送信する。Turnstile は
  # 無効化 (WIKINO_TURNSTILE_ENABLED=false) されている必要があり、でないと送信が
  # 弾かれる。
  pw_checked fill 'input[name="email"]' "$email" >/dev/null
  printf '%s' "$pass" | build_password_script
  if ! pw_checked_secret run-code --filename="$PASSWORD_SCRIPT_FILE"; then
    echo "sign-in did not complete: filling the password failed" >&2
    exit 1
  fi
  rm -f "$PASSWORD_SCRIPT_FILE"

  # The context now holds the credentials, so the on-disk config is no longer
  # needed; drop it to avoid leaving credentials at rest.
  #
  # [Ja] コンテキストが creds を保持したので、ディスク上の config はもう不要。
  # creds を残さないため削除する。
  rm -f "$CONFIG_FILE"

  local result
  result="$(sign_in_status)"

  # An account with two-factor auth enabled stops at the TOTP challenge instead
  # of completing the sign-in, so generate the current code and submit it. The
  # secret stays inside devtotp, which prints only the six-digit code. Accounts
  # without two-factor auth never reach this step and go straight through.
  #
  # [Ja] 2 要素認証が有効なアカウントはサインインを完了せず TOTP チャレンジで
  # 止まるため、現在のコードを生成して送信する。secret は devtotp の中で完結し、
  # 標準出力に出るのは 6 桁のコードだけ。2 要素認証を設定していないアカウントは
  # このステップに来ないため、そのまま通る。
  if [[ "$result" == "NOT_SIGNED_IN $origin/sign_in/two_factor/"* ]]; then
    local code
    code="$(cd "$GO_DIR" && WIKINO_DEVTOTP_EMAIL="$email" go run ./cmd/devtotp)"
    pw_checked fill 'input[name="totp_code"]' "$code" --submit >/dev/null
    result="$(sign_in_status)"
  fi

  if [[ "$result" == NOT_SIGNED_IN* ]]; then
    echo "sign-in did not complete (still under /sign_in): ${result#NOT_SIGNED_IN }" >&2
    exit 1
  fi
  if [[ "$result" != "SIGNED_IN $origin" && "$result" != "SIGNED_IN $origin/"* ]]; then
    echo "could not verify sign-in at the expected origin: $result" >&2
    exit 1
  fi
  echo "logged in as $role: ${result#SIGNED_IN }"
}

cmd_shot() {
  local path="${1:-/}"
  if [ ! -f "$ORIGIN_FILE" ]; then
    echo "no active session; run 'browse.sh login' first" >&2
    exit 1
  fi
  mkdir -p "$SHOT_DIR"
  local origin
  origin="$(cat "$ORIGIN_FILE")"
  local name
  name="$(printf '%s' "$path" | sed 's#[^a-zA-Z0-9]#_#g; s#^_*##')"
  [ -n "$name" ] || name=home

  pw_checked goto "$origin$path" >/dev/null
  pw_checked run-code "async page => page.waitForLoadState('networkidle')" >/dev/null
  pw_checked screenshot --filename="$SHOT_DIR/$name.png" >/dev/null
  echo "screenshot: $SHOT_DIR/$name.png"
}

cmd_close() {
  pw close >/dev/null 2>&1 || true
  rm -f "$CONFIG_FILE" "$PASSWORD_SCRIPT_FILE" "$CURL_CONFIG_FILE" "$ORIGIN_FILE"
  rm -rf "$LEGACY_PROFILE_DIR"
  echo "browser session closed and temp files removed"
}

main() {
  case "${1:-}" in
    check)
      shift
      cmd_check "${1:-}"
      ;;
    login)
      shift
      cmd_login "${1:-}"
      ;;
    shot)
      shift
      cmd_shot "${1:-/}"
      ;;
    close)
      cmd_close
      ;;
    *)
      echo "usage: browse.sh {check [role] | login [role] | shot <path> | close}" >&2
      exit 2
      ;;
  esac
}

# Keep the function definitions sourceable for subprocess tests without
# changing the behavior of direct invocations.
#
# [Ja] サブプロセステストで関数定義だけを source できるようにしつつ、直接実行時の
# 動作は変えない。
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
