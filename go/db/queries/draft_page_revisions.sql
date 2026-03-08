-- name: CreateDraftPageRevision :one
-- 下書きページリビジョンを作成する
INSERT INTO draft_page_revisions (draft_page_id, space_id, space_member_id, title, body, body_html, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: DeleteDraftPageRevisionsByDraftPageID :exec
-- 下書きページIDに紐づくリビジョンをすべて削除する
DELETE FROM draft_page_revisions WHERE draft_page_id = $1 AND space_id = $2;
