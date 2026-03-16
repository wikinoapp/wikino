-- name: CreateSuggestionComment :one
-- 編集提案コメントを作成する
INSERT INTO suggestion_comments (space_id, suggestion_id, created_space_member_id, body, body_html, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListSuggestionCommentsBySuggestionID :many
-- 編集提案IDでコメント一覧を取得する（作成日時の昇順）
SELECT * FROM suggestion_comments
WHERE suggestion_id = $1 AND space_id = $2
ORDER BY created_at ASC;

-- name: FindSuggestionCommentByID :one
-- IDで編集提案コメントを取得する（スペースIDでスコープ）
SELECT * FROM suggestion_comments
WHERE id = $1 AND space_id = $2;

-- name: CountSuggestionCommentsBySuggestionID :one
-- 編集提案IDでコメント数を取得する
SELECT COUNT(*) FROM suggestion_comments
WHERE suggestion_id = $1 AND space_id = $2;
