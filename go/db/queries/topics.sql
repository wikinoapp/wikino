-- name: FindTopicBySpaceAndNumber :one
-- スペースIDとナンバーでトピックを取得する（削除されていないトピックのみ）
SELECT * FROM topics WHERE space_id = $1 AND number = $2 AND discarded_at IS NULL;

-- name: ListActiveTopicsBySpace :many
-- スペースID でアクティブなトピック一覧を取得する（削除されていないトピックのみ）
SELECT * FROM topics WHERE space_id = $1 AND discarded_at IS NULL ORDER BY number;

-- name: FindTopicsBySpaceAndNames :many
-- スペースID と名前リストでトピックを取得する（削除されていないトピックのみ、Wikiリンク解析時のトピック一括検索用）
SELECT * FROM topics WHERE space_id = $1 AND name = ANY($2::varchar[]) AND discarded_at IS NULL;

-- name: FindTopicBySpaceAndID :one
-- スペースIDとIDでトピックを取得する（削除されていないトピックのみ）
SELECT * FROM topics WHERE space_id = $1 AND id = $2 AND discarded_at IS NULL;

-- name: FindTopicsByIDsAndSpace :many
-- スペースIDとIDリストでトピックを一括取得する（削除されていないトピックのみ）
SELECT * FROM topics WHERE space_id = $1 AND id = ANY($2::uuid[]) AND discarded_at IS NULL;

-- name: ListTopicsJoinedBySpaceMember :many
-- スペースメンバーが参加しているトピック一覧を取得する（編集画面のトピックセレクター用）
SELECT t.* FROM topics t
INNER JOIN topic_members tm ON t.id = tm.topic_id
WHERE tm.space_member_id = $1 AND t.space_id = $2 AND t.discarded_at IS NULL
ORDER BY t.number;
