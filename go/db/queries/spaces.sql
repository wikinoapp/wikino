-- name: GetSpaceByIdentifier :one
-- 識別子でスペースを取得する（削除されていないスペースのみ）
SELECT * FROM spaces WHERE identifier = $1 AND discarded_at IS NULL;

-- name: ListActiveSpacesByUser :many
-- ユーザーが参加中（active）かつ削除されていないスペースの一覧を取得する
-- Rails 版 current_user.active_space_records 相当
-- 並び順はユーザーがスペースに参加した日の降順（最近参加したスペースが上）
SELECT s.* FROM spaces s
INNER JOIN space_members sm ON sm.space_id = s.id
WHERE sm.user_id = $1
  AND sm.active = TRUE
  AND s.discarded_at IS NULL
ORDER BY sm.joined_at DESC;
