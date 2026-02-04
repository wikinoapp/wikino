-- name: IncrementRateLimit :one
-- Rate Limit カウンターをインクリメントする（UPSERT）
-- 同一のkey + window_startが存在する場合はcountをインクリメント、なければ新規作成
INSERT INTO rate_limits (key, window_start, count, created_at, updated_at)
VALUES ($1, $2, 1, NOW(), NOW())
ON CONFLICT (key, window_start)
DO UPDATE SET
    count = rate_limits.count + 1,
    updated_at = NOW()
RETURNING *;

-- name: GetRateLimitCount :one
-- 指定されたkeyとウィンドウ開始時刻でのカウントを取得する
SELECT count FROM rate_limits
WHERE key = $1 AND window_start = $2;

-- name: DeleteOldRateLimits :exec
-- 指定された時刻より古いRate Limitレコードを削除する
DELETE FROM rate_limits
WHERE window_start < $1;
