-- name: ListDraftPagesByUser :many
-- ホーム画面に表示する、ユーザーの下書きページ一覧を取得する。
-- draft_pages → pages → topics → spaces → space_membersをJOINし、ユーザーがアクティブな
-- スペースメンバーであるスペースに限定する。
SELECT
  dp.id AS draft_page_id,
  dp.title AS draft_page_title,
  dp.modified_at AS draft_page_modified_at,
  p.id AS page_id,
  p.title AS page_title,
  p.number AS page_number,
  t.name AS topic_name,
  t.visibility AS topic_visibility,
  s.identifier AS space_identifier,
  s.name AS space_name
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

-- name: ListDraftPagesBySpaceMember :many
-- 同一スペース内のスペースメンバー自身の下書きページ一覧を更新日時の降順で取得する (ページ編集画面の下書き一覧カラム用)。
-- draft_pages → pages → topics → spacesをJOINし、指定スペース・スペースメンバーに限定する。
-- 提案編集用の下書き (suggestion_page_id付き) も、ホーム画面の下書き一覧と同じく除外せず含める。
SELECT
  dp.id AS draft_page_id,
  dp.title AS draft_page_title,
  dp.modified_at AS draft_page_modified_at,
  p.id AS page_id,
  p.title AS page_title,
  p.number AS page_number,
  t.name AS topic_name,
  t.visibility AS topic_visibility,
  s.identifier AS space_identifier,
  s.name AS space_name
FROM draft_pages dp
INNER JOIN pages p ON dp.page_id = p.id AND dp.space_id = p.space_id
INNER JOIN topics t ON dp.topic_id = t.id AND dp.space_id = t.space_id
INNER JOIN spaces s ON dp.space_id = s.id
WHERE dp.space_member_id = $1
  AND dp.space_id = $2
  AND p.discarded_at IS NULL
  AND t.discarded_at IS NULL
  AND s.discarded_at IS NULL
ORDER BY dp.modified_at DESC
LIMIT $3;

-- name: ListDraftPagesByUserForIndex :many
-- ユーザーの下書きページ一覧を取得する (下書き一覧画面用)
-- スペース名・トピック名を含み、スペース名・トピック名の順にソート
SELECT
  dp.id AS draft_page_id,
  dp.title AS draft_page_title,
  dp.modified_at AS draft_page_modified_at,
  p.id AS page_id,
  p.title AS page_title,
  p.number AS page_number,
  t.id AS topic_id,
  t.name AS topic_name,
  t.number AS topic_number,
  t.visibility AS topic_visibility,
  s.id AS space_id,
  s.identifier AS space_identifier,
  s.name AS space_name
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
ORDER BY s.name, t.name, dp.modified_at DESC;

-- name: ListDraftPagesByModifiedAt :many
-- 公開APIの下書き一覧。スペースメンバー自身の下書きのうち、ページが未廃棄・未ゴミ箱で、
-- 対象のトピック (topic_ids) に属するものを更新日時の新しい順に返す。
-- topic_idsは呼び出し元が、閲覧者が開けて下書きを読めるトピックに解決したもので、廃棄済みのトピックを含まない。
-- トピックはWeb画面の下書き一覧と同じく下書きの所属トピックで絞る (ページを移動すると下書きの所属トピックも移る)。
-- カーソル (after_modified_at, after_id) を渡すと、並び順でその位置より後ろの下書きだけを返す。
-- 更新日時が同じ下書きも主キーで順序を決めるため、取得中に更新が無ければ重複や読み飛ばしなく辿れる。
-- メンバーの下書きは (space_member_id, page_id) のインデックスで引き、並べ替えはその件数の中で行う。
SELECT
  dp.*,
  p.number AS page_number
FROM draft_pages dp
INNER JOIN pages p ON dp.page_id = p.id AND p.space_id = @space_id
WHERE dp.space_id = @space_id
  AND dp.space_member_id = @space_member_id
  AND dp.topic_id = ANY(@topic_ids::uuid[])
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND (
    sqlc.narg('after_modified_at')::timestamp IS NULL
    OR (dp.modified_at, dp.id) < (sqlc.narg('after_modified_at')::timestamp, sqlc.narg('after_id')::uuid)
  )
ORDER BY dp.modified_at DESC, dp.id DESC
LIMIT @row_limit;
