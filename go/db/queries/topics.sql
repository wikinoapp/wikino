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

-- name: FindFirstJoinedTopicBySpaceMember :one
-- Returns the topic with the smallest id among those the space member has joined
-- (not-discarded topics only), scoped to the given space. Used by the empty-state
-- "create a new page" link on the space detail page.
--
-- [Ja] スペースメンバーが参加しているトピックのうち id が最小のもの (削除されていない
-- トピックのみ) を、指定スペースにスコープして返す。スペース詳細画面の空状態で表示する
-- 「新しいページを作る」導線で使用する。
SELECT t.* FROM topics t
INNER JOIN topic_members tm ON t.id = tm.topic_id
WHERE tm.space_member_id = $1 AND t.space_id = $2 AND t.discarded_at IS NULL
ORDER BY t.id ASC
LIMIT 1;
