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
-- 公開 og:image 配信用: 「生きている公開トピックのページからのみ参照されている」場合に限り blob 情報を返す
-- Rails 版 AttachmentRecord#all_referencing_pages_public? (= referencing_topics.any? &&
-- referencing_topics.all?(&:visibility_public?)) と等価。判定スコープからは
-- 論理削除済みのページ・トピックを除外する (生きている参照を 1 件以上持ち、かつそれらが
-- すべて visibility=0 の場合のみ blob を返す)。space スコープは取らず、URL 文字列を
-- 知っている誰でも (ゲスト含む) 閲覧可能であることを前提にする。
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
      AND p.discarded_at IS NULL
      AND t.discarded_at IS NULL
  )
  AND NOT EXISTS (
    SELECT 1 FROM page_attachment_references par
    INNER JOIN pages p ON par.page_id = p.id
    INNER JOIN topics t ON p.topic_id = t.id
    WHERE par.attachment_id = a.id
      AND p.discarded_at IS NULL
      AND t.discarded_at IS NULL
      AND t.visibility <> 0
  );
