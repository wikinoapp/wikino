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
  AND space_id = $3
  AND discarded_at IS NULL;

-- name: SearchPageLocations :many
-- ページロケーションを検索する（Wikiリンク補完用。公開済み・未廃棄・未ゴミ箱のページのみ）
SELECT p.title, t.name AS topic_name
FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.discarded_at IS NULL
WHERE p.space_id = $1
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND p.published_at IS NOT NULL
  AND p.title IS NOT NULL
  AND p.title ILIKE ALL($2::text[])
ORDER BY p.modified_at DESC
LIMIT 10;

-- name: GetNextPageNumber :one
-- スペース内の次のページ番号を取得する
SELECT COALESCE(MAX(number), 0) + 1 AS next_number FROM pages WHERE space_id = $1;

-- name: FindLinkedPagesPaginated :many
-- リンク先ページをオフセットページネーションで取得する（同スペース・公開済み・未廃棄のページのみ）
SELECT * FROM pages
WHERE id = ANY($1::uuid[])
  AND space_id = $2
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
ORDER BY modified_at DESC, id DESC
LIMIT $3
OFFSET $4;

-- name: CountLinkedPages :one
-- リンク先ページの総件数を取得する（同スペース・公開済み・未廃棄のページのみ）
SELECT COUNT(*)
FROM pages
WHERE id = ANY($1::uuid[])
  AND space_id = $2
  AND published_at IS NOT NULL
  AND discarded_at IS NULL;

-- name: FindBacklinkedPagesPaginated :many
-- バックリンクページをオフセットページネーションで取得する（同スペース・公開済み・未廃棄のページのみ）
SELECT * FROM pages
WHERE $1::varchar = ANY(linked_page_ids)
  AND space_id = $2
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
ORDER BY modified_at DESC, id DESC
LIMIT $3
OFFSET $4;

-- name: CountBacklinkedPages :one
-- バックリンクページの総件数を取得する（同スペース・公開済み・未廃棄のページのみ）
SELECT COUNT(*)
FROM pages
WHERE $1::varchar = ANY(linked_page_ids)
  AND space_id = $2
  AND published_at IS NOT NULL
  AND discarded_at IS NULL;

-- name: CreateLinkedPage :one
-- Wikiリンクから参照されるページを作成する
INSERT INTO pages (space_id, topic_id, number, title, body, body_html, linked_page_ids, modified_at, published_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, '', '', '{}', $5, $5, $5, $5)
RETURNING *;
