-- name: ListJoinedTopicsByUser :many
-- ユーザーが参加しているトピック一覧を取得する（サイドバー表示用）
-- topic_members → topics → spaces を JOIN し、アクティブなスペースメンバーのスペースに限定
SELECT
  t.id AS topic_id,
  t.number AS topic_number,
  t.name AS topic_name,
  t.visibility AS topic_visibility,
  s.id AS space_id,
  s.identifier AS space_identifier,
  s.name AS space_name
FROM topic_members tm
INNER JOIN topics t ON tm.topic_id = t.id AND t.space_id = tm.space_id
INNER JOIN spaces s ON t.space_id = s.id
INNER JOIN space_members sm ON tm.space_member_id = sm.id AND sm.space_id = tm.space_id
WHERE sm.user_id = $1
  AND sm.active = true
  AND t.discarded_at IS NULL
  AND s.discarded_at IS NULL
ORDER BY tm.last_page_modified_at DESC NULLS LAST, t.number DESC
LIMIT $2;
