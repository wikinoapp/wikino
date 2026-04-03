-- name: FindDraftPageByPageAndMember :one
-- ページIDとスペースメンバーIDで下書きを取得する
SELECT * FROM draft_pages WHERE page_id = $1 AND space_member_id = $2 AND space_id = $3;

-- name: CreateDraftPage :one
-- 下書きを作成する
INSERT INTO draft_pages (space_id, page_id, space_member_id, topic_id, suggestion_page_id, title, body, body_html, linked_page_ids, featured_image_attachment_id, modified_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: UpdateDraftPage :one
-- 下書きを更新する
UPDATE draft_pages
SET topic_id = $2,
    title = $3,
    body = $4,
    body_html = $5,
    linked_page_ids = $6,
    featured_image_attachment_id = $7,
    modified_at = $8,
    updated_at = $9
WHERE id = $1 AND space_id = $10
RETURNING *;

-- name: UpdateDraftPageSuggestionPageID :one
-- 下書きの編集提案ページIDを更新する
UPDATE draft_pages
SET suggestion_page_id = $2,
    updated_at = $3
WHERE id = $1 AND space_id = $4
RETURNING *;

-- name: FindDraftPageByID :one
-- IDで下書きを取得する（スペースIDでスコープ）
SELECT * FROM draft_pages WHERE id = $1 AND space_id = $2;

-- name: DeleteDraftPage :exec
-- 下書きを削除する
DELETE FROM draft_pages WHERE id = $1 AND space_id = $2;

-- name: FindDraftPageBySuggestionPageID :one
-- 編集提案ページIDで下書きを取得する（スペースIDでスコープ）
SELECT * FROM draft_pages WHERE suggestion_page_id = $1 AND space_id = $2;

-- name: ListDraftPagesByMemberAndTopic :many
-- スペースメンバーIDとトピックIDで下書きページ一覧を取得する（編集提案作成画面用）
SELECT
  dp.*,
  p.title AS page_title,
  p.number AS page_number
FROM draft_pages dp
INNER JOIN pages p ON dp.page_id = p.id AND dp.space_id = p.space_id
WHERE dp.space_member_id = $1
  AND dp.topic_id = $2
  AND dp.space_id = $3
  AND dp.suggestion_page_id IS NULL
  AND p.discarded_at IS NULL
ORDER BY dp.modified_at DESC;
