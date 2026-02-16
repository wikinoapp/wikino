-- name: GetSpaceByIdentifier :one
-- 識別子でスペースを取得する（削除されていないスペースのみ）
SELECT * FROM spaces WHERE identifier = $1 AND discarded_at IS NULL;
