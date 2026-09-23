#!/usr/bin/env bash
#
# browse.shはplaywright-cliを駆動してdevサイトのブラウザ確認を行う。
# Basic認証configの生成・devサインイン (アカウントが2要素認証を有効にしている
# 場合はそのステップの通過も含む)・ログイン済みセッションでのスクショ・後片付けを
# まとめる。
#
# サインインに使う資格情報は、シードがアカウントを作成する元にしている名簿
# (go/seed-users.toml) から `wikino devcreds` を通して読む。ここでサインインする
# アカウントを、シードが実際に作成したアカウントそのものにするため。ベースURLは
# 引き続きKORYLUS_BROWSING_BASE_URLを前提とするため、op runラッパー配下
# (go/Makefileのbrowse-* ターゲット) から実行する。このURLをop run経由で読む
# ことで、.envをシェル評価して、そこに含まれるBasic認証credsを壊すのを避ける
# (credsが `$` を含む場合)。devサーバはTurnstileを
# 無効化 (devの .envでWIKINO_TURNSTILE_ENABLED=false) して起動している必要が
# あり、でないとBot検証でサインインの送信が弾かれる。2要素認証を有効にした
# アカウントを使う場合は、TOTPコードをuser_two_factor_authsから生成するため
# dev DBへ到達できることも前提になる。
set -euo pipefail

SESSION=dev
# 役割の指定が無いときにサインインするアカウント。ownerは両スペースを管理し、
# フィーチャーフラグを全件持つため、最も多くの画面へ到達できる。
DEFAULT_ROLE=owner
# Goモジュールのルート。本スクリプトの位置から求めることで、どのディレクトリ
# から実行してもヘルパー (wikino devcredsとdevtotp) を解決できるようにする。
# 名簿を探す基準もここになる。wikino devcredsが名簿のパスをモジュールルートからの
# 相対で持っているため。
GO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# 資格情報を含むファイルとスクショの書き込み先。リポジトリのgitignore済み
# tmpディレクトリであり、サブプロセステストはWIKINO_BROWSE_TMP_DIRで別の場所へ
# 向ける。テストへ差し込み口を1つだけ渡すことが、以下のファイル名の定義をここ
# だけにする方法になる。テスト側で名前を書き写すと、ここで改名したときにテストは
# 通ったまま、本スクリプトが開発者の実tmpディレクトリへパスワードのファイルを
# 書くことになるため。
TMP_DIR="${WIKINO_BROWSE_TMP_DIR:-/workspace/tmp}"
CONFIG_FILE="$TMP_DIR/browse-cli.config.json"
PASSWORD_SCRIPT_FILE="$TMP_DIR/browse-cli.password.js"
ORIGIN_FILE="$TMP_DIR/browse-cli.origin"
SHOT_DIR="$TMP_DIR/browse"
LEGACY_PROFILE_DIR="$TMP_DIR/browse-cli-profile"

# playwright-cliのChromiumはE2Eとは別の専用パス
# (WIKINO_PLAYWRIGHT_CLI_BROWSERS_PATH) に焼かれている。コンテナ既定の
# PLAYWRIGHT_BROWSERS_PATH (/ms-playwright) はE2E用のChromiumを指すため、
# playwright-cli用のパスに向けて、E2Eではなく対応するChromiumビルドを
# 解決させる。
export PLAYWRIGHT_BROWSERS_PATH="${WIKINO_PLAYWRIGHT_CLI_BROWSERS_PATH:-/ms-playwright-cli}"

pw() { playwright-cli -s="$SESSION" "$@"; }

# pw_checkedは、playwright-cliが終了コード0のまま出力へ記録する
# Playwrightのコマンドレベルエラーを検出する。呼び出し側は成功時の出力を
# 捨てられるが、エラーは常にstderrへ残す。
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

# pw_checked_secretは、コードが秘密を含むコマンド用のpw_checked。応答の本文を
# 端末へ出さない。
#
# run-codeは実行したコードを "### Ran Playwright code" セクションとして呼び出し側へ
# 返すが、サインインではそのコードが、パスワードを含む生成スクリプトそのものになる。
# 現状は漏れない (成功時の応答は捨てており、playwright-cliはthrowされたエラーへ
# Errorセクションだけを返す) が、どちらもplaywright-cli側が変えられるものである。
# Errorセクションだけを流すことが、応答にセクションが増えたときにも、端末・CIログ・
# エージェントのトランスクリプトへパスワードを出さない方法になる。この関数が流すと
# 指示されていないセクションは流れない。
#
# セクションの見出しを1つも持たない出力は応答ではない。playwright-cliが何かを読む
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

# sign_in_statusは、ブラウザが現在いるページについて
# "<SIGNED_IN|NOT_SIGNED_IN> <url>" を出力する。サインインが完了すると /sign_inから
# 離れる (ホームかback URLへ)。/sign_in配下に留まる場合は未ログインを意味し、
# フォームの再描画 (422) か、2要素認証のチャレンジで止まっているかのいずれか。
# pathnameは `new URL()` ではなく正規表現で取り出す。playwright-cliのrun-codeは
# URLコンストラクタが未定義のサンドボックスで動くため。判定はsentinelで返して
# bash側で扱い、失敗時に非ゼロ終了できるようにする (run-code内のthrowはエラーを
# 表示するだけで終了コードは0になるため)。
sign_in_status() {
  local result
  result="$(pw_checked --raw run-code "async page => {
    await page.waitForLoadState('networkidle');
    const href = page.url();
    const path = href.replace(/^[a-z][a-z0-9+.-]*:\/\/[^/]+/i, '').replace(/[?#].*/, '');
    const notSignedIn = path === '/sign_in' || path.startsWith('/sign_in/');
    return (notSignedIn ? 'NOT_SIGNED_IN ' : 'SIGNED_IN ') + href;
  }")"

  # --rawは返り値の文字列を二重引用符で囲むため、判定前に取り除く。
  result="${result%\"}"
  result="${result#\"}"
  printf '%s\n' "$result"
}

# build_configはKORYLUS_BROWSING_BASE_URLからBasic認証config
# (httpCredentials) を生成し、以降の遷移用にcredsを抜いたoriginファイルも書く。
# configはcredsを含むためgitignore済みtmpに0600で書き、ログインが
# ブラウザコンテキストに取り込んだ直後に削除する。
build_config() {
  mkdir -p "$TMP_DIR"
  node -e '
    const fs = require("fs");
    const raw = process.env.KORYLUS_BROWSING_BASE_URL || "";
    if (!raw) { console.error("KORYLUS_BROWSING_BASE_URLが設定されていない"); process.exit(1); }
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

# build_password_scriptは、パスワード入力と送信の操作を0600のファイルへ
# 書く。パスワードは標準入力でNodeへ渡し、playwright-cliにはファイルパスだけを
# 渡すことで、どちらのプロセスのargvにも秘密を載せない。
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

# role_forは、指定された値を名簿が持つ役割へ読み替える。名簿はアカウントを
# 役割で名指しするが、名簿ができる前の本スクリプトはKORYLUS_BROWSING_USER{N} の組の
# 番号を受け取っていたため、当時存在した2つの番号を、それが指していた役割として
# 受け付ける。読み替えをGo側ではなくここに置いているのは、名簿にこの番号を
# 知らせないためで、互換をやめるときの変更もこの関数だけで済む。
role_for() {
  case "$1" in
    1) printf 'owner\n' ;;
    2) printf 'collaborator\n' ;;
    *) printf '%s\n' "$1" ;;
  esac
}

# CURL_CONFIG_FILEは到達確認で使うBasic認証の資格情報を持つ。curlには -uではなく0600の
# 設定ファイルから読ませ、パスワードスクリプトがplaywright-cliに対してそうしているのと同じく、
# プロセスの引数に資格情報が現れないようにする。
CURL_CONFIG_FILE="$TMP_DIR/browse-cli.curlrc"

# check_environmentは、ブラウザ確認に必要なものを、資格情報の値を出さずに検証する。検査を
# browse.shに置くことで、診断とログインが同じop runラッパーと同じ要件を通る。手で組み立てた
# 診断は要件を1つ落とすことがあり、健全な環境を壊れていると報告してしまう。
#
# 最後の手順ではdev URLからページを取得し、誰が応答したのかを見る。リバースプロキシはGo版が
# 引き受けないパスをすべてRails版へ渡すため、Go版へ到達できない状態は大きな音を立てて失敗せず、
# 同じパスをRails版が解釈した結果 (Go版にしかない画面なら404) が返ってくる。Go版が必ず設定する
# CSRFクッキーの有無が、この2つを見分ける手がかりになる。
check_environment() {
  local failed=0

  if [ "${APP_ENV:-}" != "dev" ]; then
    echo "APP_ENVはdevである必要がある (Makefileのbrowseターゲット経由で実行する)" >&2
    failed=1
  fi

  if [ -z "${KORYLUS_BROWSING_BASE_URL:-}" ]; then
    echo "KORYLUS_BROWSING_BASE_URLが設定されていない" >&2
    failed=1
  fi

  if [ "${WIKINO_TURNSTILE_ENABLED:-}" != "false" ]; then
    echo "devのブラウザログインにはWIKINO_TURNSTILE_ENABLEDをfalseにする必要がある" >&2
    failed=1
  fi

  local command_name
  for command_name in node curl playwright-cli; do
    if ! command -v "$command_name" >/dev/null 2>&1; then
      echo "${command_name}を利用できない" >&2
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
        console.error("KORYLUS_BROWSING_BASE_URLが有効なURLではない");
        process.exit(1);
      }
      if (url.protocol !== "http:" && url.protocol !== "https:") {
        console.error("KORYLUS_BROWSING_BASE_URLはhttpかhttpsである必要がある");
        process.exit(1);
      }
      if (!url.username || !url.password) {
        console.error("KORYLUS_BROWSING_BASE_URLにBasic認証の資格情報が含まれていない");
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


# check_credentialsは、尋ねている役割を名簿が持っているかを報告する。check_environmentでは
# なく診断の側に置いているのは、ログインの経路がこの直後に同じ資格情報を読み、同じ失敗を自ら
# 報告するためである。ここでも実行すると、ログインのたびに名簿へ2度尋ねることになる。
check_credentials() {
  local role
  role="$(role_for "${1:-$DEFAULT_ROLE}")"

  if ! (cd "$GO_DIR" && go run ./cmd/wikino devcreds "$role" >/dev/null 2>&1); then
    echo "役割 '$role' の資格情報が無い: wikino devcredsが読む名簿を確認する" >&2

    return 1
  fi
}

# check_reachableはdev URLからサインイン画面を取得し、どのアプリケーションが応答したかを
# 報告する。資格情報は0600の設定ファイル経由でcurlへ渡し、リクエストが終わり次第そのファイルを
# 削除する。
check_reachable() (
  # 後片付けのtrapはこの検査のsubshell内だけに置く。ログインコマンド側にある別の
  # 資格情報ファイル用trapを置き換えず、正常return・コマンド失敗・signalのすべてで実行する。
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
      console.error("KORYLUS_BROWSING_BASE_URLの資格情報に制御文字を含めることはできない");
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
    echo "dev URLが応答しなかった" >&2

    return 1
  fi

  local status="${response%% *}"
  case "$status" in
    200) ;;
    401)
      echo "dev URLがKORYLUS_BROWSING_BASE_URLのBasic認証の資格情報を拒否した" >&2

      return 1
      ;;
    *)
      echo "dev URLの/sign_inが${status}を返した" >&2

      return 1
      ;;
  esac

  # Go版はすべての応答にCSRFクッキーを設定するため、それが無いということは、リクエストが
  # Rails版へ届いたことを意味する。サインイン画面を提供するのはGo版である。
  if ! printf '%s' "$response" | grep -q "wikino_csrf_token"; then
    echo "dev URLの応答にGo版のCSRFクッキーが無い: リクエストがGo版に届いていない" >&2

    return 1
  fi
)

# cmd_checkは、ブラウザ確認に必要な環境が整っているかを報告する。失敗を、手書きの診断で
# 追いかけるのではなく、原因のある場所で読めるようにするため。
cmd_check() {
  check_environment
  check_credentials "${1:-}"
  check_reachable
  echo "ブラウザ確認の環境が整った: $(role_for "${1:-$DEFAULT_ROLE}")"
}

cmd_login() {
  local role
  role="$(role_for "${1:-$DEFAULT_ROLE}")"

  # 資格情報は、シードが読むのと同じ名簿 (go/seed-users.toml) から
  # `wikino devcreds` を通して受け取る。devcredsはメールアドレスとパスワードを
  # 1行ずつ出力する。シードが尋ねたファイルに尋ねることが、シード直後のサインインが
  # 通り続ける理由になる。片方だけを変えてもう片方が元のまま、ということが起こらない
  # ため。パスワードは全経路でargvに載せず、標準出力からNodeの標準入力を経て
  # 0600のスクリプトへ書く。playwright-cliにはそのパスだけを渡して入力直後に削除し、
  # そのスクリプトを返してくる応答はpw_checked_secretが扱う。
  local credentials
  credentials="$(cd "$GO_DIR" && go run ./cmd/wikino devcreds "$role")"
  local lines=()
  mapfile -t lines <<<"$credentials"
  local email="${lines[0]:-}"
  local pass="${lines[1]:-}"
  if [ "${#lines[@]}" -ne 2 ] || [ -z "$email" ] || [ -z "$pass" ]; then
    echo "役割 '$role' の資格情報が無い: wikino devcredsは空でない行をちょうど2行出力する必要がある" >&2
    exit 1
  fi

  # 資格情報を含むファイルをどの終了経路でも削除し、ログイン途中の失敗でも
  # Basic認証configやパスワードスクリプトをディスクに残さない。
  trap 'rm -f "$CONFIG_FILE" "$PASSWORD_SCRIPT_FILE"' EXIT

  build_config
  local origin
  origin="$(cat "$ORIGIN_FILE")"

  # Basic認証はconfig (httpCredentials) で渡す。ブラウザは永続プロファイル
  # 無しで開く。playwright-cliは名前付きセッションのブラウザをシェル呼び出しの間も
  # 生かし続けるため、ログインCookieはコンテキスト内で残る。加えて永続プロファイル
  # では最初の遷移以降ページへ入力イベントが届かなくなる (2要素認証フォームに入力は
  # できても送信できない)。
  pw_checked open "$origin/sign_in" --browser=chromium --config="$CONFIG_FILE" >/dev/null

  # Wikinoのサインインは単一ステップのフォーム (email + passwordを一括送信)。
  # emailは送信せずに入力し、passwordでEnterを押して送信する。Turnstileは
  # 無効化 (WIKINO_TURNSTILE_ENABLED=false) されている必要があり、でないと送信が
  # 弾かれる。
  pw_checked fill 'input[name="email"]' "$email" >/dev/null
  printf '%s' "$pass" | build_password_script
  if ! pw_checked_secret run-code --filename="$PASSWORD_SCRIPT_FILE"; then
    echo "サインインが完了しなかった: パスワードの入力に失敗した" >&2
    exit 1
  fi
  rm -f "$PASSWORD_SCRIPT_FILE"

  # コンテキストがcredsを保持したので、ディスク上のconfigはもう不要。
  # credsを残さないため削除する。
  rm -f "$CONFIG_FILE"

  local result
  result="$(sign_in_status)"

  # 2要素認証が有効なアカウントはサインインを完了せずTOTPチャレンジで
  # 止まるため、現在のコードを生成して送信する。secretはdevtotpの中で完結し、
  # 標準出力に出るのは6桁のコードだけ。2要素認証を設定していないアカウントは
  # このステップに来ないため、そのまま通る。
  if [[ "$result" == "NOT_SIGNED_IN $origin/sign_in/two_factor/"* ]]; then
    local code
    code="$(cd "$GO_DIR" && WIKINO_DEVTOTP_EMAIL="$email" go run ./cmd/devtotp)"
    pw_checked fill 'input[name="totp_code"]' "$code" --submit >/dev/null
    result="$(sign_in_status)"
  fi

  if [[ "$result" == NOT_SIGNED_IN* ]]; then
    echo "サインインが完了しなかった (/sign_inのまま): ${result#NOT_SIGNED_IN }" >&2
    exit 1
  fi
  if [[ "$result" != "SIGNED_IN $origin" && "$result" != "SIGNED_IN $origin/"* ]]; then
    echo "想定したoriginでサインインを確認できなかった: $result" >&2
    exit 1
  fi
  echo "${role}としてログインした: ${result#SIGNED_IN }"
}

cmd_shot() {
  local path="${1:-/}"
  if [ ! -f "$ORIGIN_FILE" ]; then
    echo "有効なセッションが無い: 先に 'browse.sh login' を実行する" >&2
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
  echo "スクリーンショット: $SHOT_DIR/$name.png"
}

cmd_close() {
  pw close >/dev/null 2>&1 || true
  rm -f "$CONFIG_FILE" "$PASSWORD_SCRIPT_FILE" "$CURL_CONFIG_FILE" "$ORIGIN_FILE"
  rm -rf "$LEGACY_PROFILE_DIR"
  echo "ブラウザのセッションを閉じ、一時ファイルを削除した"
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
      echo "使い方: browse.sh {check [role] | login [role] | shot <path> | close}" >&2
      exit 2
      ;;
  esac
}

# サブプロセステストで関数定義だけをsourceできるようにしつつ、直接実行時の
# 動作は変えない。
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
