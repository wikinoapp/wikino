-- name: FindTopicMemberBySpaceMemberAndTopic :one
-- スペースメンバーIDとトピックIDでトピックメンバーを取得する
SELECT * FROM topic_members WHERE space_member_id = $1 AND topic_id = $2 AND space_id = $3;

-- name: ListTopicMembersBySpaceMemberAndTopics :many
-- Fetch the topic memberships for the given topic ids in one query (used to avoid N+1
-- when resolving per-topic permissions on the space detail page).
-- [Ja] トピック ID リストでトピックメンバーを 1 クエリ一括取得する (スペース詳細の権限判定で N+1 を避けるために使用)。
SELECT * FROM topic_members WHERE space_member_id = $1 AND space_id = $2 AND topic_id = ANY($3::uuid[]);

-- name: ListTopicMembersByUserAndTopics :many
-- Fetch the topic memberships the given user holds across multiple topics in one query, joining
-- space_members to resolve the user (the user owns different space_members across spaces). Used to
-- avoid N+1 when resolving per-topic create permissions for joined topics spanning many spaces on
-- the home page. Scoped by space_id to satisfy the space_id query convention.
--
-- [Ja] ユーザーが複数トピックで持つトピックメンバーを 1 クエリで一括取得する (スペースごとに
-- 別々の space_member を持つため space_members と JOIN してユーザーを解決する)。ホーム画面で
-- 複数スペースにまたがる参加中トピックのページ作成権限を解決する際の N+1 を避けるために使用。
-- space_id 条件でスコープし space_id クエリ規約を満たす。
SELECT tm.* FROM topic_members tm
INNER JOIN space_members sm ON tm.space_member_id = sm.id AND tm.space_id = sm.space_id
WHERE sm.user_id = $1 AND tm.space_id = ANY($2::uuid[]) AND tm.topic_id = ANY($3::uuid[]);

-- name: UpdateTopicMemberLastPageModifiedAt :exec
-- トピックメンバーのlast_page_modified_atを更新する（ページ公開時に使用）
UPDATE topic_members SET last_page_modified_at = $1, updated_at = $2 WHERE topic_id = $3 AND space_member_id = $4 AND space_id = $5;
