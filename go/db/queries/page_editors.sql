-- name: FindPageEditorByPageAndSpaceMember :one
-- ページIDとスペースメンバーIDでページ編集者を取得する
SELECT * FROM page_editors WHERE page_id = $1 AND space_member_id = $2;

-- name: CreatePageEditor :one
-- ページ編集者を作成する
INSERT INTO page_editors (space_id, page_id, space_member_id, last_page_modified_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdatePageEditorLastPageModifiedAt :one
-- ページ編集者のlast_page_modified_atを更新する
UPDATE page_editors
SET last_page_modified_at = $2, updated_at = $3
WHERE id = $1 AND space_id = $4
RETURNING *;
