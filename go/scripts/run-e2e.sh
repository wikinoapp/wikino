#!/bin/bash
#
# E2Eテスト用サーバーを起動し、Playwrightテストを実行し、サーバーを停止するスクリプト
#
# 使い方:
#   bash scripts/run-e2e.sh ""                          # 全テスト実行
#   bash scripts/run-e2e.sh "tests/tab-indent.spec.ts"  # 特定テスト実行

set -euo pipefail

E2E_PORT=4201
E2E_HEALTH_URL="http://localhost:${E2E_PORT}/health"
E2E_MAX_WAIT=30
E2E_TEST_FILE="${1:-}"
E2E_SERVER_PID=""

cleanup() {
  if [ -n "$E2E_SERVER_PID" ]; then
    echo "==> サーバーを停止 (PID: $E2E_SERVER_PID)"
    kill "$E2E_SERVER_PID" 2>/dev/null || true
  fi

  # Free the port before waiting on the wrapper PID. `op run -- go run` does not
  # forward SIGTERM to the compiled child binary, so the child keeps holding the
  # port and the wrapper never exits; a bare `wait` on the wrapper PID then blocks
  # forever. Killing the actual port holder first lets the wrapper exit so the
  # `wait` below can reap it instead of hanging.
  #
  # [Ja] ラッパー PID を wait する前にポートを解放する。`op run -- go run` は
  # SIGTERM をコンパイル済みの子プロセスに伝播しないため、子がポートを保持し続け
  # ラッパーも終了しない。その状態でラッパー PID を素朴に wait するとハングする。
  # 先に実際のポート保持プロセスを kill するとラッパーが終了でき、下の wait は
  # ハングせずにそれを回収できる。
  fuser -k "${E2E_PORT}/tcp" 2>/dev/null || true

  if [ -n "$E2E_SERVER_PID" ]; then
    wait "$E2E_SERVER_PID" 2>/dev/null || true
  fi
}

trap cleanup EXIT

echo "==> E2Eテスト用サーバーを起動 (port=${E2E_PORT})"
APP_ENV=test_e2e op run --env-file=".env" -- go run ./cmd/wikino serve &
E2E_SERVER_PID=$!

echo "==> ヘルスチェックで起動を待機..."
waited=0
while [ $waited -lt $E2E_MAX_WAIT ]; do
  if curl -sf "$E2E_HEALTH_URL" > /dev/null 2>&1; then
    echo "==> サーバーが起動しました"
    break
  fi
  sleep 1
  waited=$((waited + 1))
done

if [ $waited -ge $E2E_MAX_WAIT ]; then
  echo "エラー: サーバーの起動がタイムアウトしました (${E2E_MAX_WAIT}秒)"
  exit 1
fi

echo "==> E2Eテストを実行"
cd e2e

if [ -n "$E2E_TEST_FILE" ]; then
  APP_ENV=test_e2e op run --env-file="../.env" -- pnpm exec playwright test "$E2E_TEST_FILE"
else
  APP_ENV=test_e2e op run --env-file="../.env" -- pnpm exec playwright test
fi
