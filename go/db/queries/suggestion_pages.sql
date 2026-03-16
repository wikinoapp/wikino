-- name: CreateSuggestionPage :one
-- 編集提案ページを作成する
INSERT INTO suggestion_pages (space_id, suggestion_id, page_id, page_revision_id, title, body, body_html, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: FindSuggestionPageByID :one
-- IDで編集提案ページを取得する（スペースIDでスコープ）
SELECT * FROM suggestion_pages WHERE id = $1 AND space_id = $2;

-- name: ListSuggestionPagesBySuggestionID :many
-- 編集提案IDで編集提案ページ一覧を取得する（作成日時の昇順）
SELECT * FROM suggestion_pages
WHERE suggestion_id = $1 AND space_id = $2
ORDER BY created_at ASC;

-- name: UpdateSuggestionPageContent :one
-- 編集提案ページのコンテンツを更新する（スペースIDでスコープ）
UPDATE suggestion_pages
SET title = $2, body = $3, body_html = $4, updated_at = $5
WHERE id = $1 AND space_id = $6
RETURNING *;
