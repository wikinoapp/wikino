-- name: ListPageAttachmentReferencesByPageID :many
-- ページIDに紐づく添付ファイル参照を取得する（スペースIDでスコープ）
SELECT par.* FROM page_attachment_references par
INNER JOIN pages p ON par.page_id = p.id
WHERE par.page_id = $1 AND p.space_id = $2;

-- name: CreatePageAttachmentReference :one
-- 添付ファイル参照を作成する
INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeletePageAttachmentReferencesByPageAndAttachmentIDs :exec
-- ページIDと添付ファイルIDリストに該当する添付ファイル参照を削除する
DELETE FROM page_attachment_references
WHERE page_id = $1 AND attachment_id = ANY($2::uuid[]);
