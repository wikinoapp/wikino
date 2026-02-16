-- name: FindPageBySpaceAndNumber :one
-- スペースIDとページ番号でページを取得する（廃棄されていないページのみ）
SELECT * FROM pages WHERE space_id = $1 AND number = $2 AND discarded_at IS NULL;

-- name: FindPagesByIDs :many
-- IDリストに含まれるページを取得する（同スペース・公開済み・未廃棄のページのみ。リンク一覧表示用）
SELECT * FROM pages
WHERE id = ANY($1::uuid[])
  AND space_id = $2
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
ORDER BY number;

-- name: FindBacklinkedPagesByPageID :many
-- linked_page_idsカラムに指定ページIDが含まれるページを取得する（同スペース・公開済み・未廃棄のページのみ。バックリンク一覧表示用）
SELECT * FROM pages
WHERE $1::varchar = ANY(linked_page_ids)
  AND space_id = $2
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
ORDER BY number;

-- name: UpdatePage :one
-- ページを更新する
UPDATE pages
SET topic_id = $2,
    title = $3,
    body = $4,
    body_html = $5,
    linked_page_ids = $6,
    modified_at = $7,
    published_at = $8,
    featured_image_attachment_id = $9,
    updated_at = $10
WHERE id = $1 AND space_id = $11
RETURNING *;

-- name: FindPageByTopicAndTitle :one
-- 指定トピック内で指定タイトルのページを取得する（廃棄されていないページのみ。Wikiリンクのページ存在確認用）
SELECT * FROM pages
WHERE topic_id = $1
  AND title = $2
  AND discarded_at IS NULL;

-- name: CreateLinkedPage :one
-- Wikiリンクから参照されるページを作成する
INSERT INTO pages (space_id, topic_id, number, title, body, body_html, linked_page_ids, modified_at, published_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, '', '', '{}', $5, $5, $5, $5)
RETURNING *;
