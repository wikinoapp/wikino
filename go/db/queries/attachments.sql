-- name: ExistsAttachmentByIDAndSpace :one
-- IDとスペースIDで添付ファイルの存在を確認する
SELECT EXISTS(
  SELECT 1 FROM attachments WHERE id = $1 AND space_id = $2
);

-- name: FindAttachmentByIDAndSpace :one
-- IDとスペースIDで添付ファイルを取得する（ファイル名を含む）
SELECT a.id, a.space_id, asb.filename
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = $1 AND a.space_id = $2;

-- name: FindAttachmentsByIDsAndSpace :many
-- IDリストとスペースIDで添付ファイルを一括取得する（バッチレンダリング用）
SELECT a.id, a.space_id, asb.filename
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = ANY($1::uuid[]) AND a.space_id = $2;

-- name: FindPubliclyReferencedAttachmentBlobByID :one
-- Returns the blob info for public og:image delivery only when the attachment is referenced
-- exclusively by live pages in public topics. Equivalent to the Rails
-- AttachmentRecord#all_referencing_pages_public? (= referencing_topics.any? &&
-- referencing_topics.all?(&:visibility_public?)): the blob is returned only when at least one
-- live reference exists and every one of them has visibility=0. The reference set excludes
-- discarded pages and topics as well as pages moved to the trash, so that the og:image of a
-- trashed page stops appearing in social link previews. Both the EXISTS and the NOT EXISTS
-- branch use the same reference set; otherwise a reference from a trashed page in a private
-- topic would still veto the visibility check. The reference set is internally constrained to
-- the attachment's space. No caller-provided space scope is needed because the endpoint assumes
-- anyone who knows the URL (guests included) may view the image once this check succeeds.
--
-- [Ja] 公開 og:image 配信用: 「生きている公開トピックのページからのみ参照されている」場合に限り
-- blob 情報を返す。Rails 版 AttachmentRecord#all_referencing_pages_public?
-- (= referencing_topics.any? && referencing_topics.all?(&:visibility_public?)) と等価で、
-- 生きている参照を 1 件以上持ち、かつそれらがすべて visibility=0 の場合のみ blob を返す。
-- 判定スコープからは論理削除済みのページ・トピックに加えてゴミ箱に入ったページも除外し、
-- ゴミ箱に入ったページの og:image が SNS のリンクプレビューに残らないようにする。EXISTS と
-- NOT EXISTS の双方で同じ参照集合を使う (揃えないと「ゴミ箱に入った非公開トピックのページ」
-- からの参照が visibility 判定に残ってしまう)。参照集合は attachment と同じ space に内部で
-- 限定する。呼び出し元から space スコープを受け取る必要はなく、この判定を通過した画像は
-- URL 文字列を知っている誰でも (ゲスト含む) 閲覧可能であることを前提にする。
SELECT a.id, a.space_id, asb.key AS blob_key, asb.content_type AS blob_content_type
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = $1
  AND EXISTS (
    SELECT 1 FROM page_attachment_references par
    INNER JOIN pages p ON par.page_id = p.id
    INNER JOIN topics t ON p.topic_id = t.id
    WHERE par.attachment_id = a.id
      AND p.space_id = a.space_id
      AND t.space_id = a.space_id
      AND p.discarded_at IS NULL
      AND p.trashed_at IS NULL
      AND t.discarded_at IS NULL
  )
  AND NOT EXISTS (
    SELECT 1 FROM page_attachment_references par
    INNER JOIN pages p ON par.page_id = p.id
    INNER JOIN topics t ON p.topic_id = t.id
    WHERE par.attachment_id = a.id
      AND p.space_id = a.space_id
      AND t.space_id = a.space_id
      AND p.discarded_at IS NULL
      AND p.trashed_at IS NULL
      AND t.discarded_at IS NULL
      AND t.visibility <> 0
  );
