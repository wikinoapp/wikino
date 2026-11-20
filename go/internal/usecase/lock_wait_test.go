package usecase

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// waitForQueryBlockedByは、queryNameのクエリを実行中の接続がblockingPIDの接続のロックを待つまで待機し、
// 待っている接続のPIDを返す。待つ前にresultの処理が終わった場合と、ctxが期限切れになった場合はテストを失敗させる
func waitForQueryBlockedBy(t *testing.T, ctx context.Context, db *sql.DB, blockingPID int, queryName string, result <-chan error) int {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var pid int
		err := db.QueryRowContext(ctx, `SELECT COALESCE((
			SELECT pid FROM pg_stat_activity
			WHERE datname = current_database() AND $1 = ANY(pg_blocking_pids(pid))
			AND query LIKE $2 LIMIT 1
		), 0)`, blockingPID, "%"+queryName+"%").Scan(&pid)
		if err != nil {
			t.Fatalf("ロック待ちの確認: %v", err)
		}
		if pid != 0 {
			return pid
		}
		select {
		case err := <-result:
			t.Fatalf("ロック待ちを確認する前に処理が終了: %v", err)
		case <-ctx.Done():
			t.Fatalf("期待した行のロックを待たなかった: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}
