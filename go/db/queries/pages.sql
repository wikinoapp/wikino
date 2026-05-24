-- name: FindPageBySpaceAndNumber :one
-- スペースIDとページ番号でページを取得する（廃棄されていないページのみ）
SELECT * FROM pages WHERE space_id = $1 AND number = $2 AND discarded_at IS NULL;

-- name: FindPagesByIDs :many
-- IDリストに含まれるページを取得する（同スペース・未廃棄のページのみ。リンク一覧表示用）
SELECT * FROM pages
WHERE id = ANY($1::uuid[])
  AND space_id = $2
  AND discarded_at IS NULL
ORDER BY number;

-- name: FindBacklinkedPagesByPageID :many
-- linked_page_idsカラムに指定ページIDが含まれるページを取得する（同スペース・未廃棄のページのみ。バックリンク一覧表示用）
SELECT * FROM pages
WHERE $1::varchar = ANY(linked_page_ids)
  AND space_id = $2
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
-- 指定トピック内で指定タイトルのページを取得する（廃棄済みを含む。Wikiリンクのページ存在確認・タイトル一意性チェック用）
SELECT * FROM pages
WHERE topic_id = $1
  AND title = $2
  AND space_id = $3;

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
-- リンク先ページをオフセットページネーションで取得する（同スペース・未廃棄のページのみ）
SELECT * FROM pages
WHERE id = ANY($1::uuid[])
  AND space_id = $2
  AND discarded_at IS NULL
ORDER BY modified_at DESC, id DESC
LIMIT $3
OFFSET $4;

-- name: CountLinkedPages :one
-- リンク先ページの総件数を取得する（同スペース・未廃棄のページのみ）
SELECT COUNT(*)
FROM pages
WHERE id = ANY($1::uuid[])
  AND space_id = $2
  AND discarded_at IS NULL;

-- name: FindBacklinkedPagesPaginated :many
-- バックリンクページをオフセットページネーションで取得する（同スペース・未廃棄のページのみ）
SELECT * FROM pages
WHERE $1::varchar = ANY(linked_page_ids)
  AND space_id = $2
  AND discarded_at IS NULL
  AND NOT (id = ANY($5::uuid[]))
ORDER BY modified_at DESC, id DESC
LIMIT $3
OFFSET $4;

-- name: CountBacklinkedPages :one
-- バックリンクページの総件数を取得する（同スペース・未廃棄のページのみ）
SELECT COUNT(*)
FROM pages
WHERE $1::varchar = ANY(linked_page_ids)
  AND space_id = $2
  AND discarded_at IS NULL
  AND NOT (id = ANY($3::uuid[]));

-- name: FindBacklinkedPagesForTargets :many
-- 複数ターゲットページのバックリンクを一括取得する（各ターゲットごとにlimit件数まで）
SELECT p.*, t.target_id
FROM unnest($1::uuid[]) AS t(target_id)
CROSS JOIN LATERAL (
  SELECT *
  FROM pages
  WHERE t.target_id::varchar = ANY(linked_page_ids)
    AND space_id = $2
    AND discarded_at IS NULL
    AND NOT (id = ANY($4::uuid[]))
  ORDER BY modified_at DESC, id DESC
  LIMIT $3
) p;

-- name: CountBacklinkedPagesForTargets :many
-- 複数ターゲットページのバックリンク件数を一括取得する
SELECT t.target_id, COUNT(p.id) AS count
FROM unnest($1::uuid[]) AS t(target_id)
LEFT JOIN pages p ON t.target_id::varchar = ANY(p.linked_page_ids)
  AND p.space_id = $2
  AND p.discarded_at IS NULL
  AND NOT (p.id = ANY($3::uuid[]))
GROUP BY t.target_id;

-- name: MovePageToTopic :one
-- ページのトピックを変更する（ページ移動）
UPDATE pages
SET topic_id = $2, updated_at = NOW()
WHERE id = $1 AND space_id = $3
RETURNING *;

-- name: FindPinnedPagesByTopic :many
-- トピック内のピン留めページを取得する（公開済み・未廃棄・未ゴミ箱のページのみ、pinned_at DESCでソート）
SELECT * FROM pages
WHERE topic_id = $1
  AND space_id = $2
  AND pinned_at IS NOT NULL
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
  AND trashed_at IS NULL
ORDER BY pinned_at DESC;

-- name: FindRegularPagesByTopicPaginated :many
-- トピック内の通常ページをオフセットページネーションで取得する（ピン留めなし・公開済み・未廃棄・未ゴミ箱のページのみ）
SELECT * FROM pages
WHERE topic_id = $1
  AND space_id = $2
  AND pinned_at IS NULL
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
  AND trashed_at IS NULL
ORDER BY modified_at DESC, id DESC
LIMIT $3
OFFSET $4;

-- name: CountRegularPagesByTopic :one
-- トピック内の通常ページの総件数を取得する（ピン留めなし・公開済み・未廃棄・未ゴミ箱のページのみ）
SELECT COUNT(*)
FROM pages
WHERE topic_id = $1
  AND space_id = $2
  AND pinned_at IS NULL
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
  AND trashed_at IS NULL;

-- name: FindPinnedPagesBySpace :many
-- Returns pinned active pages across a space (published, not discarded, not trashed, and
-- whose topic is not discarded), ordered by pinned_at DESC, id DESC. The topic join also
-- enforces the "topic not discarded" part of the active scope, which a space-wide listing
-- needs because pages span multiple topics. The join is additionally scoped by
-- t.space_id = @space_id as a defensive measure, so topic visibility is always evaluated
-- within the requested space per the space_id query-scoping rule. When public_only is true,
-- only pages in public topics (visibility = 0) are returned, for non-member viewers.
--
-- [Ja] スペース内のピン留めされたアクティブなページ (公開済み・未廃棄・未ゴミ箱・トピック
-- 未廃棄) を pinned_at DESC, id DESC で返す。トピック JOIN はアクティブ判定の「トピック未廃棄」
-- 条件も担う。スペース横断の一覧ではページが複数トピックにまたがるためこの JOIN が必要。
-- JOIN は防御的に t.space_id = @space_id でもスコープし、space_id クエリスコープのルールに従って
-- トピックの可視性を常に対象スペース内で評価する。public_only が true のときは公開トピック
-- (visibility = 0) のページのみに絞る (非メンバー閲覧者向け)。
SELECT p.* FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.space_id = @space_id
  AND p.pinned_at IS NOT NULL
  AND p.published_at IS NOT NULL
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND (@public_only::boolean IS FALSE OR t.visibility = 0)
ORDER BY p.pinned_at DESC, p.id DESC;

-- name: FindRegularPagesBySpacePaginated :many
-- Returns non-pinned active pages across a space with offset pagination, ordered by
-- modified_at DESC, id DESC. Active and public_only handling matches FindPinnedPagesBySpace.
--
-- [Ja] スペース内の通常ページ (ピン留めなし) をオフセットページネーションで取得する。
-- 並び順は modified_at DESC, id DESC。アクティブ判定と public_only の扱いは
-- FindPinnedPagesBySpace と同じ。
SELECT p.* FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.space_id = @space_id
  AND p.pinned_at IS NULL
  AND p.published_at IS NOT NULL
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND (@public_only::boolean IS FALSE OR t.visibility = 0)
ORDER BY p.modified_at DESC, p.id DESC
LIMIT @row_limit
OFFSET @row_offset;

-- name: CountRegularPagesBySpace :one
-- Returns the total count of non-pinned active pages across a space. Filtering matches
-- FindRegularPagesBySpacePaginated so the count and the page slice stay consistent.
--
-- [Ja] スペース内の通常ページ (ピン留めなし) の総件数を返す。フィルタ条件は
-- FindRegularPagesBySpacePaginated と揃えており、件数とページ一覧の整合性を保つ。
SELECT COUNT(*)
FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.space_id = @space_id
  AND p.pinned_at IS NULL
  AND p.published_at IS NOT NULL
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND (@public_only::boolean IS FALSE OR t.visibility = 0);

-- name: DiscardPageByID :exec
-- 指定ページを論理削除する（タイトルをIDに変更し、discarded_at を設定する）
UPDATE pages
SET title = id::varchar,
    discarded_at = @discarded_at,
    updated_at = @updated_at
WHERE id = @id
  AND space_id = @space_id;

-- name: CreateLinkedPage :one
-- Wikiリンクから参照されるページを作成する
INSERT INTO pages (space_id, topic_id, number, title, body, body_html, linked_page_ids, modified_at, published_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, '', '', '{}', $5, NULL, $5, $5)
RETURNING *;
