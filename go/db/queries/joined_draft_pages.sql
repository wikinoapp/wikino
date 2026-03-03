-- name: ListDraftPagesByUser :many
-- ユーザーの下書きページ一覧を取得する（サイドバー表示用）
-- draft_pages → pages → topics → spaces → space_members を JOIN し、アクティブなスペースメンバーのスペースに限定
SELECT
  dp.id AS draft_page_id,
  dp.title AS draft_page_title,
  dp.modified_at AS draft_page_modified_at,
  p.id AS page_id,
  p.title AS page_title,
  p.number AS page_number,
  t.name AS topic_name,
  t.visibility AS topic_visibility,
  s.identifier AS space_identifier
FROM draft_pages dp
INNER JOIN pages p ON dp.page_id = p.id AND dp.space_id = p.space_id
INNER JOIN topics t ON dp.topic_id = t.id AND dp.space_id = t.space_id
INNER JOIN spaces s ON dp.space_id = s.id
INNER JOIN space_members sm ON dp.space_member_id = sm.id AND dp.space_id = sm.space_id
WHERE sm.user_id = $1
  AND sm.active = true
  AND p.discarded_at IS NULL
  AND t.discarded_at IS NULL
  AND s.discarded_at IS NULL
ORDER BY dp.modified_at DESC
LIMIT $2;
