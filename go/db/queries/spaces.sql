-- name: GetSpaceByIdentifier :one
-- 識別子でスペースを取得する（削除されていないスペースのみ）
SELECT * FROM spaces WHERE identifier = $1 AND discarded_at IS NULL;

-- name: ListActiveSpacesByUser :many
-- ユーザーが参加中（active）かつ削除されていないスペースの一覧を取得する
-- Rails 版 current_user.active_space_records 相当
-- 並び順はユーザーがスペースに参加した日の降順（最近参加したスペースが上）
SELECT s.* FROM spaces s
INNER JOIN space_members sm ON sm.space_id = s.id
WHERE sm.user_id = $1
  AND sm.active = TRUE
  AND s.discarded_at IS NULL
ORDER BY sm.joined_at DESC;

-- name: GetSpaceByID :one
-- Returns the space with the given ID, discarded spaces excluded. A worker reaches a space
-- through the record it is processing, which carries the ID rather than the identifier.
--
-- [Ja] 指定 ID のスペースを返す (削除済みのスペースは除く)。ワーカーは処理対象のレコード経由で
-- スペースに到達するが、そのレコードが持っているのは識別子ではなく ID である。
SELECT * FROM spaces WHERE id = @id AND discarded_at IS NULL;

-- name: LockSpaceByID :one
-- Serializes export creation for a space until the transaction ends.
--
-- The lock is FOR NO KEY UPDATE rather than FOR UPDATE because a foreign key check takes
-- FOR KEY SHARE on the row it points at, and that conflicts with FOR UPDATE. Fifteen tables
-- reference spaces(id), so the stronger lock would make every page and topic written to the
-- space wait on a lock taken to keep two exports apart. The two weaker locks still conflict
-- with each other, which is what the serialization needs.
--
-- [Ja] トランザクションが終わるまでスペースのエクスポート作成を直列化する。
--
-- FOR UPDATE ではなく FOR NO KEY UPDATE を使うのは、外部キーの検査が参照先の行へ
-- FOR KEY SHARE を取り、それが FOR UPDATE と競合するためである。spaces(id) を参照する
-- テーブルは 15 個あり、強いほうのロックでは、2 つのエクスポートを引き離すためのロックで
-- そのスペースへのページ・トピックの書き込みまで待たされる。弱いほうのロックどうしは
-- 競合したままなので、直列化に必要なものは失われない。
SELECT id FROM spaces WHERE id = @id AND discarded_at IS NULL FOR NO KEY UPDATE;
