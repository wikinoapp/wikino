-- name: ExistsAttachmentByIDAndSpace :one
-- IDとスペースIDで添付ファイルの存在を確認する
SELECT EXISTS(
  SELECT 1 FROM attachments WHERE id = $1 AND space_id = $2
);

-- name: FindAttachmentByIDAndSpace :one
-- IDとスペースIDで添付ファイルを取得する (ファイル名を含む)
SELECT a.id, a.space_id, asb.filename
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = $1 AND a.space_id = $2;

-- name: FindAttachmentsByIDsAndSpace :many
-- IDリストとスペースIDで添付ファイルを一括取得する (バッチレンダリング用)
SELECT a.id, a.space_id, asb.filename
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = ANY($1::uuid[]) AND a.space_id = $2;

-- name: FindPubliclyReferencedAttachmentBlobByID :one
-- 公開og:image配信用: 「生きている公開トピックのページからのみ参照されている」場合に限り
-- blob情報を返す。Rails版AttachmentRecord#all_referencing_pages_public?
-- (= referencing_topics.any? && referencing_topics.all?(&:visibility_public?)) と等価で、
-- 生きている参照を1件以上持ち、かつそれらがすべてvisibility=0の場合のみblobを返す。
-- 判定スコープからは論理削除済みのページ・トピックに加えてゴミ箱に入ったページも除外し、
-- ゴミ箱に入ったページのog:imageがSNSのリンクプレビューに残らないようにする。EXISTSと
-- NOT EXISTSの双方で同じ参照集合を使う (揃えないと「ゴミ箱に入った非公開トピックのページ」
-- からの参照がvisibility判定に残ってしまう)。参照集合はattachmentと同じspaceに内部で
-- 限定する。呼び出し元からspaceスコープを受け取る必要はなく、この判定を通過した画像は
-- URL文字列を知っている誰でも (ゲスト含む) 閲覧可能であることを前提にする。
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

-- name: ListAttachmentsByPageIDsAndSpace :many
-- 指定したページが参照している添付ファイルを、参照元のページとオブジェクトのストレージ
-- キーとあわせて返す。エクスポートはその両方を使う。どのトピックのディレクトリに複製を置くかは
-- 参照元のページが決め、キーはオブジェクトの取得に使う。
--
-- 1つの添付ファイルが複数回返ることがある。スペース内の複数のページが同じ添付ファイルを参照
-- しうるためである。並び順を固定しているのは、複製の名前が毎回同じになるようにするためである。
SELECT par.page_id, a.id, a.space_id, asb.filename, asb.key AS blob_key
FROM page_attachment_references par
INNER JOIN pages p ON par.page_id = p.id
INNER JOIN attachments a ON par.attachment_id = a.id AND a.space_id = p.space_id
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE par.page_id = ANY(@page_ids::uuid[]) AND p.space_id = @space_id
ORDER BY par.page_id, a.id;
