-- name: CreatePageRevision :one
-- ページリビジョンを作成する
INSERT INTO page_revisions (space_id, space_member_id, page_id, title, body, body_html, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;
