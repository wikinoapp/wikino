#!/bin/bash
#
# 開発用のサーバーに対して、ビルドしたCLIのe2eテスト (e2e/) を実行するスクリプト。
#
# シードのスペースの管理者の個人アクセストークンをデータベースに直接発行し、
# テストの後に失効させる。サーバーとシードは起動・投入済みであることを前提にする。
# テストはページと下書きを新しく作るだけで、既存のデータを書き換えないため、
# 手元の開発用のデータベースにも実行できる (作ったページは残る)。
#
# 使い方 (DATABASE_URLはサーバーと同じデータベースを指す):
#   DATABASE_URL=... WIKINO_E2E_HOST=http://localhost:4201 bash scripts/e2e.sh

set -euo pipefail

: "${DATABASE_URL:?DATABASE_URLにサーバーと同じデータベースを指定してください}"
: "${WIKINO_E2E_HOST:?WIKINO_E2E_HOSTに開発用のサーバーのURLを指定してください}"

# シードのスペースと、その管理者のアットネーム (go/seed-users.example.tomlのowner)
E2E_SPACE="seed-wiki"
E2E_ATNAME="seeduser1"
E2E_SCOPES="topic:read,page:read,page:write,draft_page:read,draft_page:write,draft_page:delete"
E2E_TOKEN_ID=""

cleanup() {
  if [ -n "$E2E_TOKEN_ID" ]; then
    echo "==> 個人アクセストークンを失効"
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -q <<SQL || true
UPDATE personal_access_tokens SET revoked_at = now(), updated_at = now()
WHERE id = '${E2E_TOKEN_ID}'
  AND space_id = (SELECT id FROM spaces WHERE identifier = '${E2E_SPACE}');
SQL
  fi
}

trap cleanup EXIT

# サーバーと同じ形 (接頭辞 + 32バイトの乱数のbase64url) のトークンを作り、
# SHA-256のダイジェストだけを保存する (go/internal/auth/opaque_token.goを参照)。
# トークンはコマンドの引数に載せず、標準入力のSQLと環境変数で渡す。
token="wkp_$(head -c 32 /dev/urandom | base64 -w0 | tr '+/' '-_' | tr -d '=')"

echo "==> ${E2E_SPACE} の @${E2E_ATNAME} の個人アクセストークンを発行"
E2E_TOKEN_ID=$(psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -qAt <<SQL
INSERT INTO personal_access_tokens (space_id, space_member_id, name, token_digest, token_last_chars, scopes, expires_at)
SELECT s.id, sm.id, 'CLIのe2e', encode(sha256(convert_to('${token}', 'UTF8')), 'hex'), right('${token}', 4),
       string_to_array('${E2E_SCOPES}', ','), now() + interval '1 hour'
FROM spaces s
INNER JOIN space_members sm ON sm.space_id = s.id AND sm.active
INNER JOIN users u ON u.id = sm.user_id
WHERE s.identifier = '${E2E_SPACE}' AND u.atname = '${E2E_ATNAME}'
RETURNING id;
SQL
)

if [ -z "$E2E_TOKEN_ID" ]; then
  echo "エラー: ${E2E_SPACE} に @${E2E_ATNAME} のメンバーがいません。シードを投入してください (make -C go seed)" >&2
  exit 1
fi

echo "==> e2eテストを実行 (${WIKINO_E2E_HOST})"
cd "$(dirname "$0")/.."
WIKINO_E2E_SPACE="$E2E_SPACE" WIKINO_E2E_TOKEN="$token" go test -tags e2e -count=1 -v ./e2e/...
