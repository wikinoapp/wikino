-- name: ListPageAttachmentReferencesByPageID :many
-- ページIDに紐づく添付ファイル参照を取得する（スペースIDでスコープ）
SELECT par.* FROM page_attachment_references par
INNER JOIN pages p ON par.page_id = p.id
WHERE par.page_id = $1 AND p.space_id = $2;

-- name: CreatePageAttachmentReference :one
-- 添付ファイル参照を作成する（スペースIDでページと添付ファイルの所属を検証）
INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at)
SELECT $1, $2, $3, $4
FROM pages p
INNER JOIN attachments a ON a.space_id = p.space_id
WHERE p.id = $2 AND a.id = $1 AND p.space_id = $5
RETURNING *;

-- name: DeletePageAttachmentReferencesByPageAndAttachmentIDs :exec
-- ページIDと添付ファイルIDリストに該当する添付ファイル参照を削除する（スペースIDでスコープ）
DELETE FROM page_attachment_references par
USING pages p
WHERE par.page_id = p.id
  AND par.page_id = $1
  AND p.space_id = $2
  AND par.attachment_id = ANY($3::uuid[]);
