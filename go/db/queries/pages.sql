-- name: FindPageBySpaceAndNumber :one
-- スペースIDとページ番号でページを取得する (廃棄されていないページのみ)
SELECT * FROM pages WHERE space_id = $1 AND number = $2 AND discarded_at IS NULL;

-- name: FindPagesByIDs :many
-- IDリストに含まれるページを取得する (同スペース・未廃棄のページのみ。リンク一覧表示用)
SELECT * FROM pages
WHERE id = ANY($1::uuid[])
  AND space_id = $2
  AND discarded_at IS NULL
ORDER BY number;

-- name: FindBacklinkedPagesByPageID :many
-- linked_page_idsカラムに指定ページIDが含まれるページを取得する (同スペース・未廃棄のページのみ。バックリンク一覧表示用)
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
    linked_page_ids = $5,
    modified_at = $6,
    published_at = $7,
    featured_image_attachment_id = $8,
    updated_at = $9
WHERE id = $1 AND space_id = $10
RETURNING *;

-- name: FindPageByTopicAndTitle :one
-- 指定トピック内で指定タイトルのページを取得する (廃棄済みを含む。Wikiリンクのページ存在確認・タイトル一意性チェック用)
SELECT * FROM pages
WHERE topic_id = $1
  AND title = $2
  AND space_id = $3;

-- name: FindPagesByTopicAndTitlePairs :many
-- トピックIDとタイトルの組の集合でページを一括取得する (廃棄済みを含む。Wikiリンクのページ存在確認用)
-- タイトルの比較はcitextに揃え、FindPageByTopicAndTitleと同じく大文字小文字を区別しない
-- DBで一致した入力との対応を維持するため、入力タイトルも返す
-- 本文 (body) はリンクの解決に使わないため、返す列をリンク先の特定に要るものだけに絞る
SELECT pages.id, pages.topic_id, pages.number, pages.title, pairs.title::text AS requested_title FROM pages
INNER JOIN (
  SELECT unnest(@topic_ids::uuid[]) AS topic_id, unnest(@titles::citext[]) AS title
) AS pairs
  ON pages.topic_id = pairs.topic_id AND pages.title = pairs.title
WHERE pages.space_id = @space_id;

-- name: SearchPageLocations :many
-- ページロケーションを検索する (Wikiリンク補完用。未廃棄・未ゴミ箱のページのみ)。
-- ページは閲覧者が開けるトピック (visible_topic_ids) に絞る。この集合は呼び出し元がページ画面と
-- 同じCanShowTopicの規則で解決する。LIMITの前に絞るため、SQLで絞り込む。
-- 未公開ページは、開けるトピックの公開ページか呼び出したメンバー自身の下書きからリンクされているものだけを含める。
-- 入力途中のタイトルで自動作成されたページや、リンクを消したあとに残ったページを候補に出さないためである。
-- キー入力のたびに呼ばれるため、リンク元の判定はGINインデックスが効く @> で書く。
SELECT p.title, t.name AS topic_name
FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.space_id = @space_id
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND p.title IS NOT NULL
  AND p.title ILIKE ALL(@title_patterns::text[])
  AND t.discarded_at IS NULL
  AND (@all_topics_visible::boolean IS TRUE OR t.id = ANY(@visible_topic_ids::uuid[]))
  AND (
    p.published_at IS NOT NULL
    OR EXISTS (
      SELECT 1 FROM pages src
      INNER JOIN topics src_t ON src.topic_id = src_t.id AND src_t.space_id = @space_id
      WHERE src.space_id = @space_id
        AND src.published_at IS NOT NULL
        AND src.discarded_at IS NULL
        AND src.trashed_at IS NULL
        AND src_t.discarded_at IS NULL
        AND (@all_topics_visible::boolean IS TRUE OR src_t.id = ANY(@visible_topic_ids::uuid[]))
        AND src.linked_page_ids @> ARRAY[p.id::varchar]
    )
    OR EXISTS (
      SELECT 1 FROM draft_pages dp
      INNER JOIN pages dp_p ON dp.page_id = dp_p.id
        AND dp_p.space_id = @space_id
        AND dp_p.discarded_at IS NULL
        AND dp_p.trashed_at IS NULL
      WHERE dp.space_id = @space_id
        AND dp.space_member_id = @space_member_id
        AND dp.linked_page_ids @> ARRAY[p.id::varchar]
    )
  )
ORDER BY p.modified_at DESC
LIMIT 10;

-- name: GetNextPageNumber :one
-- スペース内の次のページ番号を取得する
SELECT COALESCE(MAX(number), 0) + 1 AS next_number FROM pages WHERE space_id = $1;

-- name: FindLinkedPagesPaginated :many
-- ページからのリンク先ページをオフセットページネーションで取得する。並び順は
-- modified_at DESC, id DESC。ゴミ箱に入ったページと廃棄済みトピックのページは除外し、
-- Wikiから取り除かれたページが古いWikiリンク経由で再び現れないようにする。トピックJOINは
-- 防御的にt.space_id = @space_idでもスコープし、space_idクエリスコープのルールに従って
-- トピックの可視性を常に対象スペース内で評価する。ページは閲覧者が開けるトピック
-- (visible_topic_ids) に絞る。この集合は呼び出し元がページ画面と同じCanShowTopicの規則で
-- 解決する。all_topics_visibleがtrueのときは絞り込みを行わない (全トピックを見せる
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
-- ページからのリンク先ページの総件数を返す。フィルタ条件はFindLinkedPagesPaginatedと
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
-- 指定ページへのバックリンクをオフセットページネーションで取得する。並び順は
-- modified_at DESC, id DESC。ゴミ箱・廃棄済みトピック・トピック可視性の扱いは
-- FindLinkedPagesPaginatedと同じ。exclude_page_idsは画面上の他の箇所で既に一覧している
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
-- 指定ページへのバックリンクの総件数を返す。フィルタ条件はFindBacklinkedPagesPaginatedと
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
-- 複数ターゲットページのバックリンクを一括取得する (各ターゲットごとにrow_limit件まで)。
-- リンク一覧が列挙するページごとにクエリを発行しないようにするためのもの。フィルタ条件は
-- FindBacklinkedPagesPaginatedと同じ。
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
-- 複数ターゲットページのバックリンク件数を一括取得する。可視ページの絞り込みをJOIN条件
-- ではなくサブクエリで行い、FindBacklinkedPagesForTargetsと同じフィルタをかけつつ、外側の
-- LEFT JOINがバックリンクを持たないターゲットに対して0件の行を返せるようにしている。
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
-- ページのトピックを変更する (ページ移動)
UPDATE pages
SET topic_id = $2, updated_at = NOW()
WHERE id = $1 AND space_id = $3
RETURNING *;

-- name: FindPinnedPagesByTopic :many
-- トピック内のピン留めページを取得する (公開済み・未廃棄・未ゴミ箱のページのみ、pinned_at DESCでソート)
SELECT * FROM pages
WHERE topic_id = $1
  AND space_id = $2
  AND pinned_at IS NOT NULL
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
  AND trashed_at IS NULL
ORDER BY pinned_at DESC;

-- name: FindRegularPagesByTopicPaginated :many
-- トピック内の通常ページをオフセットページネーションで取得する (ピン留めなし・公開済み・未廃棄・未ゴミ箱のページのみ)
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
-- トピック内の通常ページの総件数を取得する (ピン留めなし・公開済み・未廃棄・未ゴミ箱のページのみ)
SELECT COUNT(*)
FROM pages
WHERE topic_id = $1
  AND space_id = $2
  AND pinned_at IS NULL
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
  AND trashed_at IS NULL;

-- name: FindPinnedPagesBySpace :many
-- スペース内のピン留めされたアクティブなページ (公開済み・未廃棄・未ゴミ箱・トピック
-- 未廃棄) をpinned_at DESC, id DESCで返す。トピックJOINはアクティブ判定の「トピック未廃棄」
-- 条件も担う。スペース横断の一覧ではページが複数トピックにまたがるためこのJOINが必要。
-- JOINは防御的にt.space_id = @space_idでもスコープし、space_idクエリスコープのルールに従って
-- トピックの可視性を常に対象スペース内で評価する。public_onlyがtrueのときは公開トピック
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
-- スペース内の通常ページ (ピン留めなし) をオフセットページネーションで取得する。
-- 並び順はmodified_at DESC, id DESC。アクティブ判定とpublic_onlyの扱いは
-- FindPinnedPagesBySpaceと同じ。
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
-- スペース内の通常ページ (ピン留めなし) の総件数を返す。フィルタ条件は
-- FindRegularPagesBySpacePaginatedと揃えており、件数とページ一覧の整合性を保つ。
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

-- name: TrashPageByID :exec
-- trashed_atを打刻してページをゴミ箱へ入れる。これはUI上のゴミ箱で、バッチ処理が使う
-- discarded_atの論理削除とは区別する。ゴミ箱に入ったページはタイトルと本文を保持したままで、
-- ゴミ箱画面から復元できる。ゴミ箱への移動はページの状態変更なのでupdated_atも併せて更新する。
-- 既にゴミ箱に入ったページに対して実行した場合は両方の時刻が更新されるだけになる。
UPDATE pages
SET trashed_at = @trashed_at,
    updated_at = @updated_at
WHERE id = @id
  AND space_id = @space_id;

-- name: DiscardPageByID :exec
-- 指定ページを論理削除する (タイトルをIDに変更し、discarded_atを設定する)
UPDATE pages
SET title = id::varchar,
    discarded_at = @discarded_at,
    updated_at = @updated_at
WHERE id = @id
  AND space_id = @space_id;

-- name: CreateUnpublishedPage :one
-- 未公開のページを作成する。Wikiリンクの解決とページ新規作成の入口は同じ列をINSERTする
-- ため、このクエリを共有する。titleをnullableにしているのは、Wikiリンクの解決では
-- タイトルが先に決まるのに対し、新規作成の入口では未定であり、ユーザーが公開するまで
-- タイトルの無いページのままになるため。ユーザーが明示的に公開するまで未公開状態を保ち、
-- 公開済みページの一覧から除外するため、published_atはNULLのままにする。
INSERT INTO pages (space_id, topic_id, number, title, body, linked_page_ids, modified_at, published_at, created_at, updated_at)
VALUES (@space_id, @topic_id, @number, sqlc.narg('title'), '', '{}', @modified_at, NULL, @modified_at, @modified_at)
RETURNING *;

-- name: ListActivePagesBySpace :many
-- スペース内のアクティブなページ (公開済み・未廃棄・未ゴミ箱・トピック未廃棄) をすべて、
-- トピック順・ページ番号順で返す。エクスポートはこの順序でアーカイブを書き出すため、同じ
-- スペースからは毎回同じアーカイブができる。リトライが先頭からやり直せるのはこのためである。
SELECT p.* FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.space_id = @space_id
  AND p.published_at IS NOT NULL
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
ORDER BY t.number, p.number;
