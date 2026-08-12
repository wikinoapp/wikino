#!/usr/bin/env bash
#
# browse.sh drives playwright-cli for browser verification of the dev site:
# it generates the Basic-auth config, runs the dev sign-in (clearing the
# two-factor step when the account has it enabled), reuses the logged-in
# session for screenshots, and cleans up.
#
# It expects KORYLUS_BROWSING_* in the environment, so run it under the op-run
# wrapper (see the browse-* targets in go/Makefile). Reading credentials through
# op-run avoids evaluating the .env in a shell, which would corrupt any
# credential containing a `$`. The dev server must run with Turnstile disabled
# (WIKINO_TURNSTILE_ENABLED=false in the dev .env) or bot verification blocks
# the sign-in submit. Accounts with two-factor auth enabled additionally need
# the dev database to be reachable, since the TOTP code is generated from
# user_two_factor_auths.
#
# [Ja] browse.sh は playwright-cli を駆動して dev サイトのブラウザ確認を行う。
# Basic 認証 config の生成・dev サインイン (アカウントが 2 要素認証を有効にしている
# 場合はそのステップの通過も含む)・ログイン済みセッションでのスクショ・後片付けを
# まとめる。
#
# KORYLUS_BROWSING_* が環境にある前提なので、op run ラッパー配下 (go/Makefile の
# browse-* ターゲット) から実行する。creds を op run 経由で読むことで、.env を
# シェル評価して `$` を含む creds を壊すのを避ける。dev サーバは Turnstile を
# 無効化 (dev の .env で WIKINO_TURNSTILE_ENABLED=false) して起動している必要が
# あり、でないと Bot 検証でサインインの送信が弾かれる。2 要素認証を有効にした
# アカウントを使う場合は、TOTP コードを user_two_factor_auths から生成するため
# dev DB へ到達できることも前提になる。
set -euo pipefail

SESSION=dev
# The Go module root, derived from this script's location so the devtotp helper
# resolves no matter which directory the script is invoked from.
#
# [Ja] Go モジュールのルート。本スクリプトの位置から求めることで、どのディレクトリ
# から実行しても devtotp ヘルパーを解決できるようにする。
GO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR=/workspace/tmp
CONFIG_FILE="$TMP_DIR/browse-cli.config.json"
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

cmd_login() {
  local n="${1:-1}"
  local email_var="KORYLUS_BROWSING_USER${n}_EMAIL"
  local pass_var="KORYLUS_BROWSING_USER${n}_PASSWORD"
  local email="${!email_var:-}"
  local pass="${!pass_var:-}"
  if [ -z "$email" ] || [ -z "$pass" ]; then
    echo "USER${n} credentials are not set (${email_var} / ${pass_var})" >&2
    exit 1
  fi

  # Remove the credential-bearing config on any exit, so a mid-login failure
  # (a set -e abort before the explicit rm below) never leaves credentials at
  # rest.
  #
  # [Ja] creds を含む config をどの終了経路でも削除し、ログイン途中の失敗
  # (下の明示 rm へ到達する前の set -e abort) でも creds をディスクに残さない。
  trap 'rm -f "$CONFIG_FILE"' EXIT

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
  pw_checked fill 'input[name="password"]' "$pass" --submit >/dev/null

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
  echo "logged in as USER${n}: ${result#SIGNED_IN }"
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
  rm -f "$CONFIG_FILE" "$ORIGIN_FILE"
  rm -rf "$LEGACY_PROFILE_DIR"
  echo "browser session closed and temp files removed"
}

case "${1:-}" in
  login)
    shift
    cmd_login "${1:-1}"
    ;;
  shot)
    shift
    cmd_shot "${1:-/}"
    ;;
  close)
    cmd_close
    ;;
  *)
    echo "usage: browse.sh {login [user_number] | shot <path> | close}" >&2
    exit 2
    ;;
esac
