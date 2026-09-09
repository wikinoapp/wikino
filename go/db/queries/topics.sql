-- name: FindTopicBySpaceAndNumber :one
-- スペースIDとナンバーでトピックを取得する（削除されていないトピックのみ）
SELECT * FROM topics WHERE space_id = $1 AND number = $2 AND discarded_at IS NULL;

-- name: ListActiveTopicsBySpace :many
-- スペースID でアクティブなトピック一覧を取得する（削除されていないトピックのみ）
SELECT * FROM topics WHERE space_id = $1 AND discarded_at IS NULL ORDER BY number;

-- name: ListPublicTopicsBySpace :many
-- Returns the active public topics in the given space (not-discarded, visibility = public),
-- ordered by number. Used by the topic section shown to non-members (guests) on the space
-- detail page, where only public topics are visible.
--
-- [Ja] 指定スペース内のアクティブな公開トピック (未廃棄・visibility = public) を number 順で返す。
-- スペース詳細画面で非メンバー (ゲスト) に表示するトピックセクションで使用し、ここでは公開
-- トピックのみが見える。
SELECT * FROM topics
WHERE space_id = $1 AND visibility = 0 AND discarded_at IS NULL
ORDER BY number;

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

-- name: GetNextTopicNumber :one
-- Returns the next topic number in the space.
--
-- [Ja] スペース内の次のトピック番号を返す。
SELECT COALESCE(MAX(number), 0) + 1 AS next_number FROM topics WHERE space_id = @space_id;

-- name: ExistsTopicBySpaceAndName :one
-- Reports whether the space already holds a topic of that name, discarded topics included. The
-- unique index on (space_id, name) covers discarded rows as well, so a name a discarded topic
-- still carries cannot be given to a new one.
--
-- [Ja] 同じ名前のトピックがそのスペースに既にあるかを返す (削除済みのトピックも含む)。
-- (space_id, name) の一意インデックスは削除済みの行も対象にするため、削除済みのトピックが
-- 持ったままの名前は新しいトピックには付けられない。
SELECT EXISTS (
    SELECT 1 FROM topics WHERE space_id = @space_id AND name = @name
) AS topic_exists;

-- name: CreateTopic :one
-- Creates a topic.
--
-- [Ja] トピックを作成する。
INSERT INTO topics (space_id, number, name, description, visibility, created_at, updated_at)
VALUES (@space_id, @number, @name, @description, @visibility, @now, @now)
RETURNING *;

-- name: ExistsTopicBySpaceAndNameExcludingID :one
-- Reports whether another topic of the space already holds that name, discarded topics included.
-- The topic given by @excluded_id is left out, so that a topic keeping its own name is not refused.
--
-- [Ja] そのスペースの別のトピックが同じ名前を既に持っているかを返す (削除済みのトピックも含む)。
-- @excluded_id のトピックは除外し、自身の名前をそのままにしたトピックが拒否されないようにする。
SELECT EXISTS (
    SELECT 1 FROM topics WHERE space_id = @space_id AND name = @name AND id <> @excluded_id
) AS topic_exists;

-- name: UpdateTopic :one
-- Updates the general settings of a topic.
--
-- [Ja] トピックの一般設定を更新する。
UPDATE topics
SET name = @name, description = @description, visibility = @visibility, updated_at = @now
WHERE id = @id AND space_id = @space_id
RETURNING *;
