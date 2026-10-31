-- name: FindPageBySpaceAndNumber :one
-- スペースIDとページ番号でページを取得する (廃棄されていないページのみ)
SELECT * FROM pages WHERE space_id = $1 AND number = $2 AND discarded_at IS NULL;

-- name: FindPageByIDForUpdate :one
-- APIの更新で、`If-Match` の照合から更新までの間に他の更新が割り込まないよう、ページの行を
-- トランザクションが終わるまでロックして取得する (廃棄されていないページのみ)。
-- FOR UPDATEではなくFOR NO KEY UPDATEを使うのは、LockSpaceByIDと同じく、pages(id) を参照する
-- テーブル (下書き・リビジョンなど) への書き込みの外部キー検査を待たせないためである。
-- UPDATEもFOR NO KEY UPDATEを取るため、同じページの更新どうしは直列化される。
SELECT * FROM pages WHERE id = @id AND space_id = @space_id AND discarded_at IS NULL FOR NO KEY UPDATE;

-- name: LockPagesByIDs :exec
-- 公開対象を先にロックし、並行するリンクの付け替えが終わってからリンクの記録を読む。
-- 複数ページの編集提案でも同じ順でロックできるよう、ID順に一括で取得する。
SELECT id FROM pages
WHERE id = ANY(@ids::uuid[]) AND space_id = @space_id AND discarded_at IS NULL
ORDER BY id
FOR NO KEY UPDATE;

-- name: FindPageByIDForShare :one
-- 下書きにページのトピックを書くまでの間にページの移動が割り込まないよう、ページの行を
-- トランザクションが終わるまで共有ロックして取得する。
-- 共有ロックどうしは競合しないため、同じページの下書きを複数のメンバーが同時に保存しても待たない。
-- 下書きに書くトピックを読むためだけに使うため、廃棄・ゴミ箱の状態では絞り込まない。
SELECT * FROM pages WHERE id = @id AND space_id = @space_id FOR SHARE;

-- name: FindPagesByIDs :many
-- IDリストに含まれるページを取得する (同スペース・未廃棄のページのみ。リンク一覧表示用)
SELECT * FROM pages
WHERE id = ANY($1::uuid[])
  AND space_id = $2
  AND discarded_at IS NULL
ORDER BY number;

-- name: FindPageMetadataByIDs :many
-- Wikiリンクのプレビューの応答と可視性判定に使う列だけを取得する。
-- リンク先の本文サイズに転送量が比例しないよう、本文は取得しない。
SELECT id, space_id, topic_id, number, title, published_at, trashed_at, discarded_at
FROM pages
WHERE id = ANY(@ids::uuid[])
  AND space_id = @space_id
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

-- name: DiscardEmptyUnpublishedPageByID :one
-- APIの作成時に、検証後も同じタイトルの空の未公開ページである場合だけ置き換える
UPDATE pages
SET title = id::varchar,
    discarded_at = @discarded_at,
    updated_at = @updated_at
WHERE id = @id
  AND space_id = @space_id
  AND topic_id = @topic_id
  AND title = @title
  AND published_at IS NULL
  AND body = ''
  AND discarded_at IS NULL
RETURNING id;

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

-- name: ListPagesByModifiedAt :many
-- 公開APIのページ一覧。未廃棄・未ゴミ箱で、対象のトピック (topic_ids) のページを更新日時の新しい順に返す。
-- topic_idsは閲覧者が開けるトピック (visible_topic_ids) を呼び出し元がtopic_numberで絞ったもので、
-- visible_topic_idsは呼び出し元がCanShowTopicの規則で解決する。
-- include_published・include_unpublishedで公開済み・未公開のページを含めるかを選ぶ。
-- 未公開ページは、SearchPageLocationsと同じく開けるトピックの公開ページか呼び出したメンバー自身の
-- 下書きからリンクされているものだけを含める。リンク元はtopic_numberで絞らず、開けるトピック全体から探す。
-- visible_topic_idsは廃棄済みのトピックを含まないため、リンク元ではトピックをJOINしない。
-- リンク元の判定はSearchPageLocationsと違い、候補ごとの @> ではなく、リンク先のIDの集合を1度だけ作る
-- 相関の無いINで書く。候補ごとの @> ではGINインデックスが選ばれず、未公開ページの数だけ
-- リンク元を走査し直して遅くなるためである。
-- titleを渡すとそのタイトルのページに絞る (citextの比較のため大文字小文字を区別しない)。
-- keyword_patternsはキーワードごとのILIKEのパターンで、すべてのパターンについてタイトルか本文の
-- どちらかに一致するページに絞る (AND検索)。空の配列なら絞らない。本文にはインデックスが無く、
-- スペース・トピック・公開状態で絞った行を走査する。タイトルの無いページも本文で一致させるため、
-- タイトルはCOALESCEで空文字列にする (NULLのままではNOT EXISTSの判定がNULLになり、一致しない
-- ページまで含めてしまう)。
-- modified_sinceを渡すとその日時以降に更新されたページに絞る。カーソル (after_modified_at,
-- after_id) を渡すと、並び順でその位置より後ろのページだけを返す。更新日時が同じページも
-- 主キーで順序を決めるため、取得中に更新が無ければ重複や読み飛ばしなく辿れる。
-- 並び順は (space_id, modified_at, id) のインデックスに合わせてある。
SELECT p.* FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.space_id = @space_id
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND t.id = ANY(@topic_ids::uuid[])
  AND (
    (@include_published::boolean IS TRUE AND p.published_at IS NOT NULL)
    OR (
      @include_unpublished::boolean IS TRUE
      AND p.published_at IS NULL
      AND (
        p.id::varchar IN (
          SELECT unnest(src.linked_page_ids) FROM pages src
          WHERE src.space_id = @space_id
            AND src.topic_id = ANY(@visible_topic_ids::uuid[])
            AND src.published_at IS NOT NULL
            AND src.discarded_at IS NULL
            AND src.trashed_at IS NULL
        )
        OR p.id::varchar IN (
          SELECT unnest(dp.linked_page_ids) FROM draft_pages dp
          INNER JOIN pages dp_p ON dp.page_id = dp_p.id
            AND dp_p.space_id = @space_id
            AND dp_p.discarded_at IS NULL
            AND dp_p.trashed_at IS NULL
          WHERE dp.space_id = @space_id
            AND dp.space_member_id = @space_member_id
        )
      )
    )
  )
  AND (sqlc.narg('title')::text IS NULL OR p.title = sqlc.narg('title')::citext)
  AND NOT EXISTS (
    SELECT 1 FROM unnest(@keyword_patterns::text[]) AS kw(pattern)
    WHERE NOT (COALESCE(p.title, '') ILIKE kw.pattern OR p.body ILIKE kw.pattern)
  )
  AND (sqlc.narg('modified_since')::timestamp IS NULL OR p.modified_at >= sqlc.narg('modified_since')::timestamp)
  AND (
    sqlc.narg('after_modified_at')::timestamp IS NULL
    OR (p.modified_at, p.id) < (sqlc.narg('after_modified_at')::timestamp, sqlc.narg('after_id')::uuid)
  )
ORDER BY p.modified_at DESC, p.id DESC
LIMIT @row_limit;

-- name: ListLinkedPagesByModifiedAt :many
-- 公開APIのページのリンク一覧。リンク元のページの保存済みのlinked_page_ids (page_ids) が指すページを、
-- 更新日時の新しい順にカーソル方式で返す。ゴミ箱・廃棄済みのページと廃棄済みのトピックのページを除き、
-- 閲覧者が開けるトピック (visible_topic_ids) のページに絞る。この絞り込みはWeb画面のリンク一覧
-- (FindLinkedPagesPaginated) と揃えてあり、未公開のページも含める。visible_topic_idsは呼び出し元が
-- CanShowTopicの規則で解決する。カーソル (after_modified_at, after_id) の扱いはListPagesByModifiedAtと同じ。
SELECT p.* FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.id = ANY(@page_ids::uuid[])
  AND p.space_id = @space_id
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND t.id = ANY(@visible_topic_ids::uuid[])
  AND (
    sqlc.narg('after_modified_at')::timestamp IS NULL
    OR (p.modified_at, p.id) < (sqlc.narg('after_modified_at')::timestamp, sqlc.narg('after_id')::uuid)
  )
ORDER BY p.modified_at DESC, p.id DESC
LIMIT @row_limit;

-- name: ListBacklinkedPagesByModifiedAt :many
-- 公開APIのページのバックリンク一覧。保存済みのlinked_page_idsでpage_idのページにリンクしているページを、
-- 更新日時の新しい順にカーソル方式で返す。ゴミ箱・廃棄済み・トピックの可視性の扱いは
-- ListLinkedPagesByModifiedAtと同じで、Web画面のバックリンク一覧 (FindBacklinkedPagesPaginated) と揃えてある。
-- リンク元の判定はWeb画面の `= ANY(linked_page_ids)` と同じ意味だが、linked_page_idsのGINインデックスを
-- 使えるよう @> で書く。
SELECT p.* FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.space_id = @space_id
WHERE p.linked_page_ids @> ARRAY[@page_id::varchar]
  AND p.space_id = @space_id
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND t.discarded_at IS NULL
  AND t.id = ANY(@visible_topic_ids::uuid[])
  AND (
    sqlc.narg('after_modified_at')::timestamp IS NULL
    OR (p.modified_at, p.id) < (sqlc.narg('after_modified_at')::timestamp, sqlc.narg('after_id')::uuid)
  )
ORDER BY p.modified_at DESC, p.id DESC
LIMIT @row_limit;

-- name: ReassignPageLinkedPageID :exec
-- ページのリンクの記録で、old_page_idのページへのリンクをnew_page_idのページへ付け替える
-- 配列の順序を保ち、付け替え先がすでに含まれる行では付け替え元を取り除くだけにして重複させない。
-- 本文とmodified_at・updated_atは変えない (版と並び順を変えないため)
UPDATE pages
SET linked_page_ids = CASE
    WHEN linked_page_ids @> ARRAY[@new_page_id::varchar] THEN array_remove(linked_page_ids, @old_page_id::varchar)
    ELSE array_replace(linked_page_ids, @old_page_id::varchar, @new_page_id::varchar)
  END
WHERE space_id = @space_id
  AND linked_page_ids @> ARRAY[@old_page_id::varchar];
