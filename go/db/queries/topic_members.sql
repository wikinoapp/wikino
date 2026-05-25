-- name: FindTopicMemberBySpaceMemberAndTopic :one
-- スペースメンバーIDとトピックIDでトピックメンバーを取得する
SELECT * FROM topic_members WHERE space_member_id = $1 AND topic_id = $2 AND space_id = $3;

-- name: ListTopicMembersBySpaceMemberAndTopics :many
-- Fetch the topic memberships for the given topic ids in one query (used to avoid N+1
-- when resolving per-topic permissions on the space detail page).
-- [Ja] トピック ID リストでトピックメンバーを 1 クエリ一括取得する (スペース詳細の権限判定で N+1 を避けるために使用)。
SELECT * FROM topic_members WHERE space_member_id = $1 AND space_id = $2 AND topic_id = ANY($3::uuid[]);

-- name: UpdateTopicMemberLastPageModifiedAt :exec
-- トピックメンバーのlast_page_modified_atを更新する（ページ公開時に使用）
UPDATE topic_members SET last_page_modified_at = $1, updated_at = $2 WHERE topic_id = $3 AND space_member_id = $4 AND space_id = $5;
