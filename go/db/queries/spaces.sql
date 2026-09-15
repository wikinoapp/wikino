-- name: GetSpaceByIdentifier :one
-- 識別子でスペースを取得する (削除されていないスペースのみ)
SELECT * FROM spaces WHERE identifier = $1 AND discarded_at IS NULL;

-- name: ListActiveSpacesByUser :many
-- ユーザーが参加中 (active) かつ削除されていないスペースの一覧を取得する
-- Rails版current_user.active_space_records相当
-- 並び順はユーザーがスペースに参加した日の降順 (最近参加したスペースが上)
SELECT s.* FROM spaces s
INNER JOIN space_members sm ON sm.space_id = s.id
WHERE sm.user_id = $1
  AND sm.active = TRUE
  AND s.discarded_at IS NULL
ORDER BY sm.joined_at DESC;

-- name: GetSpaceByID :one
-- 指定IDのスペースを返す (削除済みのスペースは除く)。ワーカーは処理対象のレコード経由で
-- スペースに到達するが、そのレコードが持っているのは識別子ではなくIDである。
SELECT * FROM spaces WHERE id = @id AND discarded_at IS NULL;

-- name: LockSpaceByID :one
-- トランザクションが終わるまでスペースのエクスポート作成を直列化する。
--
-- FOR UPDATEではなくFOR NO KEY UPDATEを使うのは、外部キーの検査が参照先の行へ
-- FOR KEY SHAREを取り、それがFOR UPDATEと競合するためである。spaces(id) を参照する
-- テーブルは15個あり、強いほうのロックでは、2つのエクスポートを引き離すためのロックで
-- そのスペースへのページ・トピックの書き込みまで待たされる。弱いほうのロックどうしは
-- 競合したままなので、直列化に必要なものは失われない。
SELECT id FROM spaces WHERE id = @id AND discarded_at IS NULL FOR NO KEY UPDATE;
