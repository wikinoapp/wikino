-- name: CreateDraftPageRevision :one
-- 下書きページリビジョンを作成する
INSERT INTO draft_page_revisions (draft_page_id, space_id, space_member_id, title, body, body_html, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;
