-- name: FindDraftPageByPageAndMember :one
-- ページIDとスペースメンバーIDで下書きを取得する
SELECT * FROM draft_pages WHERE page_id = $1 AND space_member_id = $2 AND space_id = $3;

-- name: CreateDraftPage :one
-- 下書きを作成する
INSERT INTO draft_pages (space_id, page_id, space_member_id, topic_id, title, body, body_html, linked_page_ids, modified_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: UpdateDraftPage :one
-- 下書きを更新する
UPDATE draft_pages
SET topic_id = $2,
    title = $3,
    body = $4,
    body_html = $5,
    linked_page_ids = $6,
    modified_at = $7,
    updated_at = $8
WHERE id = $1 AND space_id = $9
RETURNING *;

-- name: DeleteDraftPage :exec
-- 下書きを削除する
DELETE FROM draft_pages WHERE id = $1 AND space_id = $2;
