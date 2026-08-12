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
-- Returns the pages linked from a page with offset pagination, ordered by modified_at DESC,
-- id DESC. Trashed pages and pages whose topic is discarded are excluded, so a page removed
-- from the wiki never resurfaces through a stale wiki link. The topic join is scoped by
-- t.space_id = @space_id as a defensive measure, so topic visibility is always evaluated
-- within the requested space per the space_id query-scoping rule. Pages are narrowed down to
-- visible_topic_ids, the topics the viewer may open, which the caller resolves with the same
-- CanShowTopic rule the page screens use. all_topics_visible skips that narrowing for
-- member-only screens that show every topic.
--
-- [Ja] ページからのリンク先ページをオフセットページネーションで取得する。並び順は
-- modified_at DESC, id DESC。ゴミ箱に入ったページと廃棄済みトピックのページは除外し、
-- Wiki から取り除かれたページが古い Wiki リンク経由で再び現れないようにする。トピック JOIN は
-- 防御的に t.space_id = @space_id でもスコープし、space_id クエリスコープのルールに従って
-- トピックの可視性を常に対象スペース内で評価する。ページは閲覧者が開けるトピック
-- (visible_topic_ids) に絞る。この集合は呼び出し元がページ画面と同じ CanShowTopic の規則で
-- 解決する。all_topics_visible が true のときは絞り込みを行わない (全トピックを見せる
-- メンバー専用画面向け)。
SELECT p.* FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.id = ANY(@page_ids::uuid[])
  AND p.space_id = @space_id
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND (@all_topics_visible::boolean IS TRUE OR t.id = ANY(@visible_topic_ids::uuid[]))
ORDER BY p.modified_at DESC, p.id DESC
LIMIT @row_limit
OFFSET @row_offset;

-- name: CountLinkedPages :one
-- Returns the total count of pages linked from a page. Filtering matches
-- FindLinkedPagesPaginated so the count and the page slice stay consistent.
--
-- [Ja] ページからのリンク先ページの総件数を返す。フィルタ条件は FindLinkedPagesPaginated と
-- 揃えており、件数とページ一覧の整合性を保つ。
SELECT COUNT(*)
FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.id = ANY(@page_ids::uuid[])
  AND p.space_id = @space_id
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND (@all_topics_visible::boolean IS TRUE OR t.id = ANY(@visible_topic_ids::uuid[]));

-- name: FindBacklinkedPagesPaginated :many
-- Returns the pages linking to a page with offset pagination, ordered by modified_at DESC,
-- id DESC. Trash, discarded-topic and topic-visibility handling matches
-- FindLinkedPagesPaginated. exclude_page_ids drops pages already listed elsewhere on the screen
-- (the page itself and its link list entries).
--
-- [Ja] 指定ページへのバックリンクをオフセットページネーションで取得する。並び順は
-- modified_at DESC, id DESC。ゴミ箱・廃棄済みトピック・トピック可視性の扱いは
-- FindLinkedPagesPaginated と同じ。exclude_page_ids は画面上の他の箇所で既に一覧している
-- ページ (ページ自身とそのリンク一覧) を除外する。
SELECT p.* FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE @page_id::varchar = ANY(p.linked_page_ids)
  AND p.space_id = @space_id
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND (@all_topics_visible::boolean IS TRUE OR t.id = ANY(@visible_topic_ids::uuid[]))
  AND NOT (p.id = ANY(@exclude_page_ids::uuid[]))
ORDER BY p.modified_at DESC, p.id DESC
LIMIT @row_limit
OFFSET @row_offset;

-- name: CountBacklinkedPages :one
-- Returns the total count of pages linking to a page. Filtering matches
-- FindBacklinkedPagesPaginated so the count and the page slice stay consistent.
--
-- [Ja] 指定ページへのバックリンクの総件数を返す。フィルタ条件は FindBacklinkedPagesPaginated と
-- 揃えており、件数とページ一覧の整合性を保つ。
SELECT COUNT(*)
FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE @page_id::varchar = ANY(p.linked_page_ids)
  AND p.space_id = @space_id
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND (@all_topics_visible::boolean IS TRUE OR t.id = ANY(@visible_topic_ids::uuid[]))
  AND NOT (p.id = ANY(@exclude_page_ids::uuid[]));

-- name: FindBacklinkedPagesForTargets :many
-- Returns the backlinks of several target pages at once (up to row_limit per target), to keep
-- the link list from issuing one query per listed page. Filtering matches
-- FindBacklinkedPagesPaginated.
--
-- [Ja] 複数ターゲットページのバックリンクを一括取得する (各ターゲットごとに row_limit 件まで)。
-- リンク一覧が列挙するページごとにクエリを発行しないようにするためのもの。フィルタ条件は
-- FindBacklinkedPagesPaginated と同じ。
SELECT p.*, targets.target_id
FROM unnest(@target_ids::uuid[]) AS targets(target_id)
CROSS JOIN LATERAL (
  SELECT pg.*
  FROM pages pg
  INNER JOIN topics t ON pg.topic_id = t.id AND t.space_id = @space_id
  WHERE targets.target_id::varchar = ANY(pg.linked_page_ids)
    AND pg.space_id = @space_id
    AND pg.discarded_at IS NULL
    AND pg.trashed_at IS NULL
    AND t.discarded_at IS NULL
    AND (@all_topics_visible::boolean IS TRUE OR t.id = ANY(@visible_topic_ids::uuid[]))
    AND NOT (pg.id = ANY(@exclude_page_ids::uuid[]))
  ORDER BY pg.modified_at DESC, pg.id DESC
  LIMIT @row_limit
) p;

-- name: CountBacklinkedPagesForTargets :many
-- Returns the backlink count of several target pages at once. The visible pages are narrowed
-- down in a subquery rather than in the join condition, so that the same filtering as
-- FindBacklinkedPagesForTargets applies while the outer LEFT JOIN still yields a zero row for
-- a target with no backlinks.
--
-- [Ja] 複数ターゲットページのバックリンク件数を一括取得する。可視ページの絞り込みを JOIN 条件
-- ではなくサブクエリで行い、FindBacklinkedPagesForTargets と同じフィルタをかけつつ、外側の
-- LEFT JOIN がバックリンクを持たないターゲットに対して 0 件の行を返せるようにしている。
SELECT targets.target_id, COUNT(p.id) AS count
FROM unnest(@target_ids::uuid[]) AS targets(target_id)
LEFT JOIN (
  SELECT pg.id, pg.linked_page_ids
  FROM pages pg
  INNER JOIN topics t ON pg.topic_id = t.id AND t.space_id = @space_id
  WHERE pg.space_id = @space_id
    AND pg.discarded_at IS NULL
    AND pg.trashed_at IS NULL
    AND t.discarded_at IS NULL
    AND (@all_topics_visible::boolean IS TRUE OR t.id = ANY(@visible_topic_ids::uuid[]))
    AND NOT (pg.id = ANY(@exclude_page_ids::uuid[]))
) p ON targets.target_id::varchar = ANY(p.linked_page_ids)
GROUP BY targets.target_id;

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
