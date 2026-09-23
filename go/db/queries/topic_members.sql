-- name: FindTopicMemberBySpaceMemberAndTopic :one
-- スペースメンバーIDとトピックIDでトピックメンバーを取得する
SELECT * FROM topic_members WHERE space_member_id = $1 AND topic_id = $2 AND space_id = $3;

-- name: ListTopicMembersBySpaceMemberAndTopics :many
-- トピックIDリストでトピックメンバーを1クエリ一括取得する (スペース詳細の権限判定でN+1を避けるために使用)。
SELECT * FROM topic_members WHERE space_member_id = $1 AND space_id = $2 AND topic_id = ANY($3::uuid[]);

-- name: ListTopicMembersByUserAndTopics :many
-- ユーザーが複数トピックで持つトピックメンバーを1クエリで一括取得する (スペースごとに
-- 別々のspace_memberを持つためspace_membersとJOINしてユーザーを解決する)。ホーム画面で
-- 複数スペースにまたがる参加中トピックのページ作成権限を解決する際のN+1を避けるために使用。
-- space_id条件でスコープしspace_idクエリ規約を満たす。
SELECT tm.* FROM topic_members tm
INNER JOIN space_members sm ON tm.space_member_id = sm.id AND tm.space_id = sm.space_id
WHERE sm.user_id = $1 AND tm.space_id = ANY($2::uuid[]) AND tm.topic_id = ANY($3::uuid[]);

-- name: UpdateTopicMemberLastPageModifiedAt :exec
-- トピックメンバーのlast_page_modified_atを更新する (ページ公開時に使用)
UPDATE topic_members SET last_page_modified_at = $1, updated_at = $2 WHERE topic_id = $3 AND space_member_id = $4 AND space_id = $5;

-- name: CreateTopicMember :one
-- スペースメンバーをトピックに参加させる。スコープは空のままにし、そのトピックでの権限は
-- メンバーがスペースに対して持つスコープから決まるようにする。
INSERT INTO topic_members (space_id, topic_id, space_member_id, joined_at, created_at, updated_at)
VALUES (@space_id, @topic_id, @space_member_id, @now, @now, @now)
RETURNING *;
