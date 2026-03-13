-- name: CreateSuggestion :one
-- 編集提案を作成する
INSERT INTO suggestions (space_id, topic_id, created_space_member_id, title, body, body_html, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: FindSuggestionByID :one
-- IDで編集提案を取得する（スペースIDでスコープ）
SELECT * FROM suggestions WHERE id = $1 AND space_id = $2;

-- name: ListSuggestionsByTopicAndStatuses :many
-- トピックIDとステータスリストで編集提案一覧を取得する（作成日時の降順）
SELECT * FROM suggestions
WHERE topic_id = $1 AND space_id = $2 AND status = ANY($3::integer[])
ORDER BY created_at DESC;

-- name: UpdateSuggestionStatus :one
-- 編集提案のステータスを更新する（スペースIDでスコープ）
UPDATE suggestions
SET status = $2, applied_at = $3, updated_at = $4
WHERE id = $1 AND space_id = $5
RETURNING *;

-- name: CountSuggestionsByTopicAndStatuses :one
-- トピックIDとステータスリストで編集提案の件数を取得する
SELECT COUNT(*)
FROM suggestions
WHERE topic_id = $1 AND space_id = $2 AND status = ANY($3::integer[]);
