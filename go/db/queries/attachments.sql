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

-- name: FindAttachmentBlobByIDAndSpace :one
-- IDとスペースIDで添付ファイルを取得する (ファイル名に加えて、種類・バイト数・ストレージキーを含む)。
-- 公開APIの添付ファイルの取得が、メタデータの応答と署名付きURLの生成に使う
SELECT a.id, a.space_id, asb.filename, asb.content_type AS blob_content_type, asb.byte_size AS blob_byte_size, asb.key AS blob_key
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = $1 AND a.space_id = $2;

-- name: FindAttachmentBlobByID :one
-- IDだけで添付ファイルを取得する (ファイル名・種類・バイト数・ストレージキーを含む)。
-- Web画面の `/attachments/{id}` はURLにスペースを含まないため、最初にこのクエリで添付ファイルと
-- そのスペースを引く。space_idを条件に持たないのはこの経路のためで、閲覧の判定は呼び出し側が
-- 引いたスペースで行う。論理削除したスペースは、物理削除の完了前でもURLを発行しない
SELECT a.id, a.space_id, asb.filename, asb.content_type AS blob_content_type, asb.byte_size AS blob_byte_size, asb.key AS blob_key
FROM attachments a
INNER JOIN spaces s ON s.id = a.space_id AND s.discarded_at IS NULL
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = $1;

-- name: FindAttachmentBlobsByIDs :many
-- IDリストだけで添付ファイルを一括取得する (ファイル名・種類・バイト数・ストレージキーを含む)。
-- Web画面の署名付きURLの一括取得 (`POST /attachments/signed_urls`) は、複数のスペースのIDを受け取りうるため、
-- 最初にこのクエリで添付ファイルとそのスペースを引く。space_idを条件に持たないのはこの経路のためで、
-- 閲覧の判定は呼び出し側が添付ファイルごとのスペースで行う。論理削除したスペースのものは除く
SELECT a.id, a.space_id, asb.filename, asb.content_type AS blob_content_type, asb.byte_size AS blob_byte_size, asb.key AS blob_key
FROM attachments a
INNER JOIN spaces s ON s.id = a.space_id AND s.discarded_at IS NULL
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = ANY(@ids::uuid[]);

-- name: ListPubliclyReferencedAttachmentIDsByIDAndSpacePairs :many
-- 取得済みのIDとスペースIDの組のうち、生きている公開トピックのページだけが参照する添付ファイルのIDを返す。
-- IDとスペースIDの配列の同じ位置を1つの組として扱い、署名付きURLの一括取得の公開判定に使う。
-- 参照集合の条件は公開og:image配信 (FindPubliclyReferencedAttachmentBlobByID) と揃える
SELECT DISTINCT a.id
FROM attachments a
INNER JOIN (
  SELECT unnest(@ids::uuid[]) AS id, unnest(@space_ids::uuid[]) AS space_id
) AS pairs ON a.id = pairs.id AND a.space_id = pairs.space_id
WHERE EXISTS (
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

-- name: FindAttachmentsByIDsAndSpace :many
-- IDリストとスペースIDで添付ファイルを一括取得する (バッチレンダリング用)
SELECT a.id, a.space_id, asb.filename
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = ANY($1::uuid[]) AND a.space_id = $2;

-- name: FindPubliclyReferencedAttachmentBlobByID :one
-- 公開og:image配信用: 「生きている公開トピックのページからのみ参照されている」場合に限り
-- blob情報を返す。生きている参照を1件以上持ち、かつそれらがすべてvisibility=0の場合のみblobを返す。
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

-- name: FindPubliclyReferencedAttachmentBlobByIDAndSpace :one
-- IDとスペースIDで、公開トピックの生きているページだけが参照する添付ファイルのblob情報を取得する。
-- 添付ファイルの所属スペースを確定した後の公開判定に使う。参照集合の条件は公開og:image配信と揃える
SELECT a.id, a.space_id, asb.key AS blob_key, asb.content_type AS blob_content_type
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = $1 AND a.space_id = $2
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

-- name: FindAttachmentForProcessingByIDAndSpace :one
-- IDとスペースIDで、アップロード後の処理に使う添付ファイルを取得する (処理の状態・作った時刻・blobの種類とストレージキーを含む)
SELECT a.id, a.space_id, a.processing_status, a.attached_at, asb.content_type AS blob_content_type, asb.byte_size AS blob_byte_size, asb.key AS blob_key
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = @id AND a.space_id = @space_id;

-- name: UpdateAttachmentProcessingStatus :execrows
-- 添付ファイルの処理の状態を、今の状態がfrom_statusのときだけto_statusに変える。
-- 変えた行の数を返し、ほかの処理が先に状態を変えていたら0になる
UPDATE attachments
SET processing_status = @to_status, updated_at = @updated_at
WHERE id = @id AND space_id = @space_id AND processing_status = @from_status;

-- name: UpdateAttachmentBlobContent :exec
-- 添付ファイルのblobのバイト数とチェックサムを、アップロード後の処理で書き換えた内容の値に更新する。
-- active_storage_blobsはspace_idを持たないため、添付ファイルを通してスペースを確かめる
UPDATE active_storage_blobs asb
SET byte_size = @byte_size, checksum = @checksum
FROM active_storage_attachments asa
INNER JOIN attachments a ON a.active_storage_attachment_id = asa.id
WHERE asb.id = asa.blob_id AND a.id = @id AND a.space_id = @space_id;

-- name: CreateAttachmentBlob :one
-- 事前署名した添付ファイルのblobを作る。Rails期に作られた行 (ActiveStorage::Blob.create_before_direct_upload!) と同じ形にする
INSERT INTO active_storage_blobs (key, filename, content_type, metadata, service_name, byte_size, checksum, created_at)
VALUES (@key, @filename, @content_type, @metadata, @service_name, @byte_size, @checksum, @created_at)
RETURNING id;

-- name: CreateAttachmentStorageAttachment :one
-- 事前署名した添付ファイルのblobを、Rails期に作られた行と同じくスペースのレコードに関連付ける
INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
VALUES ('file', 'SpaceRecord', @space_id, @blob_id, @created_at)
RETURNING id;

-- name: CreateAttachment :one
-- 事前署名した添付ファイルを、処理待ちの状態で作る
INSERT INTO attachments (space_id, active_storage_attachment_id, attached_space_member_id, attached_at, created_at, updated_at, processing_status)
VALUES (@space_id, @active_storage_attachment_id, @attached_space_member_id, @attached_at, @attached_at, @attached_at, @processing_status)
RETURNING id;

-- name: ListAttachmentObjectKeysByIDsAndSpace :many
-- 指定した添付ファイルの実体のキーを返す。添付ファイルのblobのキーと、Rails期に作られたサムネイル (variant) の
-- blobのキーである。variantのblobは、active_storage_variant_recordsを指すactive_storage_attachmentsからたどる
SELECT asb.key
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = ANY(@ids::uuid[]) AND a.space_id = @space_id
UNION ALL
SELECT vb.key
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_variant_records vr ON vr.blob_id = asa.blob_id
INNER JOIN active_storage_attachments vasa ON vasa.record_type = 'ActiveStorage::VariantRecord' AND vasa.record_id = vr.id
INNER JOIN active_storage_blobs vb ON vasa.blob_id = vb.id
WHERE a.id = ANY(@ids::uuid[]) AND a.space_id = @space_id;

-- name: DeleteAttachmentVariantsByIDsAndSpace :many
-- 指定した添付ファイルのサムネイル (variant) の行を消す。active_storage_variant_records・それを指す
-- active_storage_attachments・variantのblobである。添付ファイルのblobを指すvariantの行が残っていると
-- blobを消せないため、DeleteAttachmentsByIDsAndSpaceより先に実行する
WITH variant_records AS (
    SELECT vr.id
    FROM active_storage_variant_records vr
    INNER JOIN active_storage_attachments asa ON vr.blob_id = asa.blob_id
    INNER JOIN attachments a ON a.active_storage_attachment_id = asa.id
    WHERE a.id = ANY(@ids::uuid[]) AND a.space_id = @space_id
), deleted_variant_attachments AS (
    DELETE FROM active_storage_attachments vasa
    USING variant_records v
    WHERE vasa.record_type = 'ActiveStorage::VariantRecord' AND vasa.record_id = v.id
    RETURNING vasa.blob_id
), deleted_variant_records AS (
    DELETE FROM active_storage_variant_records vr
    USING variant_records v
    WHERE vr.id = v.id
)
DELETE FROM active_storage_blobs b
USING deleted_variant_attachments d
WHERE b.id = d.blob_id
RETURNING b.key;

-- name: DeleteAttachmentsByIDsAndSpace :many
-- 指定した添付ファイルの行を消す。page_attachment_references・attachments・active_storage_attachments・
-- active_storage_blobsである。カバー画像にしていたページ・下書き・編集提案のページの列は、外部キーがNULLにする
WITH targets AS (
    SELECT a.id, a.active_storage_attachment_id, asa.blob_id
    FROM attachments a
    INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
    WHERE a.id = ANY(@ids::uuid[]) AND a.space_id = @space_id
), deleted_references AS (
    DELETE FROM page_attachment_references par
    USING targets t
    WHERE par.attachment_id = t.id
), deleted_attachments AS (
    DELETE FROM attachments a
    USING targets t
    WHERE a.id = t.id AND a.space_id = @space_id
), deleted_storage_attachments AS (
    DELETE FROM active_storage_attachments asa
    USING targets t
    WHERE asa.id = t.active_storage_attachment_id
)
DELETE FROM active_storage_blobs b
USING targets t
WHERE b.id = t.blob_id
RETURNING b.key;

-- name: CountAttachmentsBySpace :one
-- スペースの添付ファイルの件数を返す。filename_patternが空でなければ、ファイル名がそのILIKEのパターンに
-- 一致するものだけを数える。添付ファイルの管理の画面のページ送りに使う
SELECT COUNT(*)
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.space_id = @space_id
  AND (@filename_pattern::text = '' OR asb.filename ILIKE @filename_pattern::text);

-- name: ListAttachmentsBySpacePaginated :many
-- スペースの添付ファイルを、アップロードの新しい順 (同じ時刻はIDの降順) に返す (ファイル名・種類・バイト数・
-- ストレージキー・作った時刻を含む)。filename_patternの扱いはCountAttachmentsBySpaceと同じにする
SELECT a.id, a.space_id, asb.filename, asb.content_type AS blob_content_type, asb.byte_size AS blob_byte_size, asb.key AS blob_key, a.attached_at
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.space_id = @space_id
  AND (@filename_pattern::text = '' OR asb.filename ILIKE @filename_pattern::text)
ORDER BY a.attached_at DESC, a.id DESC
LIMIT @row_limit
OFFSET @row_offset;

-- name: ListAttachmentIDsBySpaceAfter :many
-- スペースの添付ファイルのIDを、IDの昇順にafter_idより後ろからrow_limit件返す。スペースの物理削除が、
-- 添付ファイルを一定の件数ずつ読んで実体と行を消すのに使う
SELECT id FROM attachments
WHERE space_id = @space_id AND id > @after_id::uuid
ORDER BY id
LIMIT @row_limit;
