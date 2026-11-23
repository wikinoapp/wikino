-- name: CreatePageRevision :one
-- ページリビジョンを作成する
INSERT INTO page_revisions (space_id, space_member_id, page_id, title, body, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: FindLatestPageRevisionByPage :one
-- ページの最新リビジョンを取得する (スペースIDでスコープ)
SELECT * FROM page_revisions WHERE page_id = $1 AND space_id = $2 ORDER BY created_at DESC LIMIT 1;

-- name: FindPageRevisionByID :one
-- ページリビジョンをIDで取得する (スペースIDでスコープ)
SELECT * FROM page_revisions WHERE id = $1 AND space_id = $2;

-- name: DeletePageRevisionsByPageIDs :exec
-- ページのリビジョンを物理削除する。リビジョンを指す編集提案のページの参照は
-- 外部キーのON DELETE SET NULLで外れる。
DELETE FROM page_revisions WHERE space_id = @space_id AND page_id = ANY(@page_ids::uuid[]);
