-- name: ListJoinedTopicsByUser :many
-- ホーム画面に表示する、ユーザーが参加しているトピック一覧を取得する。
-- topic_members → topics → spacesをJOINし、アクティブなスペースメンバーのスペースに限定。
-- 並び順はtopic_members.last_page_modified_atの降順 (NULLS LAST)、同点はトピック番号の降順。
-- 「自分の作業視点」で並べたいため、自分が直近にページ公開操作を行ったトピックを上に出す。
-- ただし以下のトレードオフは許容する:
--   - topic_members.last_page_modified_atは公開操作を行った本人のtopic_memberしか更新
--     されない (publish_page.go / apply_suggestion.go参照)。閲覧や他メンバーの編集では
--     順序が動かない。
--   - まだ自分が公開操作をしていないトピックはNULLとなり末尾 (NULLS LAST) に並ぶ。NULL
--     同士はトピック番号の大きい方が上に来る。
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
