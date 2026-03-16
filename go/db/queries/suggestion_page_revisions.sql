-- name: CreateSuggestionPageRevision :one
-- 編集提案ページリビジョンを作成する
INSERT INTO suggestion_page_revisions (space_id, suggestion_page_id, editor_space_member_id, title, body, body_html, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListSuggestionPageRevisionsBySuggestionPageID :many
-- 編集提案ページIDでリビジョン一覧を取得する（作成日時の昇順）
SELECT * FROM suggestion_page_revisions
WHERE suggestion_page_id = $1 AND space_id = $2
ORDER BY created_at ASC;

-- name: FindLatestSuggestionPageRevision :one
-- 編集提案ページの最新リビジョンを取得する（スペースIDでスコープ）
SELECT * FROM suggestion_page_revisions
WHERE suggestion_page_id = $1 AND space_id = $2
ORDER BY created_at DESC
LIMIT 1;
