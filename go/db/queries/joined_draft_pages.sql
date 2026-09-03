-- name: ListDraftPagesByUser :many
-- Returns the user's draft pages for the home page.
-- Joins draft_pages → pages → topics → spaces → space_members and limits results to
-- spaces in which the user is an active member.
--
-- [Ja] ホーム画面に表示する、ユーザーの下書きページ一覧を取得する。
-- draft_pages → pages → topics → spaces → space_members を JOIN し、ユーザーがアクティブな
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
-- Fetch a space member's own draft pages within a single space, newest first (for the page editor's draft list column).
-- Joins draft_pages → pages → topics → spaces, scoped to the given space and space member.
-- Suggestion-edit drafts (those with suggestion_page_id set) are intentionally included,
-- matching the home page's draft list behavior.
--
-- [Ja] 同一スペース内のスペースメンバー自身の下書きページ一覧を更新日時の降順で取得する (ページ編集画面の下書き一覧カラム用)。
-- draft_pages → pages → topics → spaces を JOIN し、指定スペース・スペースメンバーに限定する。
-- 提案編集用の下書き (suggestion_page_id 付き) も、ホーム画面の下書き一覧と同じく除外せず含める。
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
-- ユーザーの下書きページ一覧を取得する（下書き一覧画面用）
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
