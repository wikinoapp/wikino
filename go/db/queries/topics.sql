-- name: FindTopicBySpaceAndNumber :one
-- スペースIDとナンバーでトピックを取得する (削除されていないトピックのみ)
SELECT * FROM topics WHERE space_id = $1 AND number = $2 AND discarded_at IS NULL;

-- name: ListActiveTopicsBySpace :many
-- スペースIDでアクティブなトピック一覧を取得する (削除されていないトピックのみ)
SELECT * FROM topics WHERE space_id = $1 AND discarded_at IS NULL ORDER BY number;

-- name: ListPublicTopicsBySpace :many
-- 指定スペース内のアクティブな公開トピック (未廃棄・visibility = public) をnumber順で返す。
-- スペース詳細画面で非メンバー (ゲスト) に表示するトピックセクションで使用し、ここでは公開
-- トピックのみが見える。
SELECT * FROM topics
WHERE space_id = $1 AND visibility = 0 AND discarded_at IS NULL
ORDER BY number;

-- name: FindTopicsBySpaceAndNames :many
-- スペースIDと名前リストでトピックを取得する (削除されていないトピックのみ、Wikiリンク解析時のトピック一括検索用)
SELECT * FROM topics WHERE space_id = $1 AND name = ANY($2::varchar[]) AND discarded_at IS NULL;

-- name: FindTopicBySpaceAndID :one
-- スペースIDとIDでトピックを取得する (削除されていないトピックのみ)
SELECT * FROM topics WHERE space_id = $1 AND id = $2 AND discarded_at IS NULL;

-- name: FindTopicsByIDsAndSpace :many
-- スペースIDとIDリストでトピックを一括取得する (削除されていないトピックのみ)
SELECT * FROM topics WHERE space_id = $1 AND id = ANY($2::uuid[]) AND discarded_at IS NULL;

-- name: ListTopicsJoinedBySpaceMember :many
-- スペースメンバーが参加しているトピック一覧を取得する (編集画面のトピックセレクター用)
SELECT t.* FROM topics t
INNER JOIN topic_members tm ON t.id = tm.topic_id
WHERE tm.space_member_id = $1 AND t.space_id = $2 AND t.discarded_at IS NULL
ORDER BY t.number;

-- name: FindFirstJoinedTopicBySpaceMember :one
-- スペースメンバーが参加しているトピックのうちidが最小のもの (削除されていない
-- トピックのみ) を、指定スペースにスコープして返す。スペース詳細画面の空状態で表示する
-- 「新しいページを作る」導線で使用する。
SELECT t.* FROM topics t
INNER JOIN topic_members tm ON t.id = tm.topic_id
WHERE tm.space_member_id = $1 AND t.space_id = $2 AND t.discarded_at IS NULL
ORDER BY t.id ASC
LIMIT 1;

-- name: GetNextTopicNumber :one
-- スペース内の次のトピック番号を返す。
SELECT COALESCE(MAX(number), 0) + 1 AS next_number FROM topics WHERE space_id = @space_id;

-- name: ExistsTopicBySpaceAndName :one
-- 同じ名前のトピックがそのスペースに既にあるかを返す (削除済みのトピックも含む)。
-- (space_id, name) の一意インデックスは削除済みの行も対象にするため、削除済みのトピックが
-- 持ったままの名前は新しいトピックには付けられない。
SELECT EXISTS (
    SELECT 1 FROM topics WHERE space_id = @space_id AND name = @name
) AS topic_exists;

-- name: CreateTopic :one
-- トピックを作成する。
INSERT INTO topics (space_id, number, name, description, visibility, created_at, updated_at)
VALUES (@space_id, @number, @name, @description, @visibility, @now, @now)
RETURNING *;

-- name: ExistsTopicBySpaceAndNameExcludingID :one
-- そのスペースの別のトピックが同じ名前を既に持っているかを返す (削除済みのトピックも含む)。
-- @excluded_idのトピックは除外し、自身の名前をそのままにしたトピックが拒否されないようにする。
SELECT EXISTS (
    SELECT 1 FROM topics WHERE space_id = @space_id AND name = @name AND id <> @excluded_id
) AS topic_exists;

-- name: UpdateTopic :one
-- トピックの一般設定を更新する。
UPDATE topics
SET name = @name, description = @description, visibility = @visibility, updated_at = @now
WHERE id = @id AND space_id = @space_id
RETURNING *;
