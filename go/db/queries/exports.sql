-- name: CreateExport :one
-- Creates an export in the given status. The repository passes queued, and the worker
-- advances it from there.
--
-- [Ja] 渡された状態でエクスポートを作成する。リポジトリは queued を渡し、そこから先は
-- ワーカーが進める。
INSERT INTO exports (space_id, queued_by_id, status, status_changed_at, created_at, updated_at)
VALUES (@space_id, @queued_by_id, @status, @now::timestamptz, @now, @now)
RETURNING *;

-- name: FindExportByIDAndSpace :one
-- Returns the export with the given ID, scoped to the space.
--
-- [Ja] 指定 ID のエクスポートを、スペースにスコープして返す。
SELECT * FROM exports WHERE id = @id AND space_id = @space_id;

-- name: FindLatestExportBySpace :one
-- Returns the most recent export of the space. The export screen shows this one, and the
-- start of a new export is refused while it is still running.
--
-- [Ja] スペースの最新のエクスポートを返す。エクスポート画面はこれを表示し、これが実行中の
-- 間は新しいエクスポートの開始を拒否する。
SELECT * FROM exports
WHERE space_id = @space_id
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListOlderExportsBySpace :many
-- Returns the exports older than the given one, oldest first. Used after a successful
-- export to delete the ones it replaces, together with their objects. The same ordering
-- as FindLatestExportBySpace keeps a later replacement out even if an older worker comes
-- back and finishes after it was considered stale.
--
-- [Ja] 指定したエクスポートより古い、同じスペースのエクスポートを古い順に返す。
-- エクスポートの成功後に、それが置き換えるエクスポートをオブジェクトごと削除するために使う。
-- FindLatestExportBySpace と同じ順序を使うことで、停止したと見なされた古いワーカーが後から完了
-- しても、その後に作られたエクスポートを削除対象に含めない。
SELECT older.*
FROM exports AS older
INNER JOIN exports AS current
    ON current.id = @current_id AND current.space_id = @space_id
WHERE older.space_id = @space_id
  AND (older.created_at, older.id) < (current.created_at, current.id)
ORDER BY older.created_at, older.id;

-- name: UpdateExportStatus :one
-- Advances the export to the given status, but only from one of the expected statuses.
-- A worker that was declared stopped and then came back to life must not overwrite the
-- state of the export that replaced it, so the transition is conditional rather than a
-- plain write. Returns no row when the current status is not one of the expected ones.
--
-- heartbeat_at and object_key keep their current value when the parameter is NULL, so
-- that a transition which has nothing to say about them does not clear them.
--
-- stale_heartbeat_before adds a second condition, for the caller that fails an export it has
-- decided is stopped: the transition is refused when the worker has reported itself alive since
-- that export was read. Without it, a worker that came back between the read and the write would
-- be failed while it is still writing its archive.
--
-- stale_status_changed_before is the same idea for an export that has no heartbeat to go by,
-- which is what a queued one waiting for a worker is. The transition is refused when the export
-- has moved on since it was read, so a job that reached a worker in the meantime is not failed
-- underneath the worker that just started it.
--
-- [Ja] 期待する状態のいずれかにある場合に限り、エクスポートを指定の状態へ進める。停止したと
-- 判断された後で生き返ったワーカーが、自分を置き換えたエクスポートの状態を上書きしないよう、
-- 遷移は単なる書き込みではなく条件付きにする。現在の状態が期待する状態のいずれでもない場合は
-- 行を返さない。
--
-- heartbeat_at と object_key はパラメータが NULL のとき現在の値を保つ。これらについて言うことの
-- 無い遷移が、値を消してしまわないようにするためである。
--
-- stale_heartbeat_before は、停止したと判断したエクスポートを失敗させる呼び出し元のために条件を
-- もう 1 つ足す。そのエクスポートを読んだ後にワーカーが生存を報告していた場合、遷移は拒否される。
-- この条件が無いと、読み取りと書き込みの間に復帰したワーカーが、アーカイブを書いている最中に
-- 失敗させられてしまう。
--
-- stale_status_changed_before は、参照できる heartbeat を持たないエクスポート、つまりワーカーを
-- 待っている queued のための同じ仕組み。読み取った後にエクスポートが先へ進んでいた場合、遷移は
-- 拒否される。その間にジョブがワーカーへ届いていたら、開始したばかりのワーカーの足元で
-- 失敗させないためである。
UPDATE exports
SET
    status = @status,
    status_changed_at = @now::timestamptz,
    heartbeat_at = COALESCE(sqlc.narg('heartbeat_at'), heartbeat_at),
    object_key = COALESCE(sqlc.narg('object_key'), object_key),
    updated_at = @now
WHERE id = @id
  AND space_id = @space_id
  AND status = ANY(@expected_statuses::integer[])
  AND (
    sqlc.narg('stale_heartbeat_before')::timestamptz IS NULL
    OR heartbeat_at IS NULL
    OR heartbeat_at < sqlc.narg('stale_heartbeat_before')
  )
  AND (
    sqlc.narg('stale_status_changed_before')::timestamptz IS NULL
    OR status_changed_at < sqlc.narg('stale_status_changed_before')
  )
RETURNING *;

-- name: UpdateExportHeartbeat :one
-- Records that the worker is still running. The status condition keeps a worker that has
-- already been replaced from making a finished export look alive again. The ID comes back
-- so that such a worker learns it was replaced and can stop, instead of finding out only
-- when its final transition is refused. No row is returned when the condition does not hold.
--
-- [Ja] ワーカーがまだ動いていることを記録する。状態の条件は、既に置き換えられたワーカーが、
-- 完了したエクスポートを生きているように見せてしまうのを防ぐ。ID を返すのは、置き換えられた
-- ワーカーが最後の遷移を拒否されるまで気付かず走り続けるのではなく、その場で置き換えを知って
-- 処理を打ち切れるようにするためである。条件を満たさない場合は行を返さない。
UPDATE exports
SET heartbeat_at = @now::timestamptz, updated_at = @now
WHERE id = @id AND space_id = @space_id AND status = @status
RETURNING id;

-- name: DeleteExport :exec
-- Deletes the export record. The ZIP object it points at is deleted by the caller.
--
-- [Ja] エクスポートのレコードを削除する。参照している ZIP オブジェクトの削除は呼び出し側が行う。
DELETE FROM exports WHERE id = @id AND space_id = @space_id;

-- name: ListLegacyExportFiles :many
-- Finds Rails ZIPs before deleting their export. Shared blobs remain in storage.
--
-- [Ja] エクスポートを削除する前に Rails の ZIP を取得する。共有 blob はストレージに残す。
SELECT DISTINCT b.key, EXISTS (
    SELECT 1 FROM active_storage_attachments other
    WHERE other.blob_id = b.id
      AND NOT (other.record_type = 'ExportRecord' AND other.record_id = e.id AND other.name = 'file')
) AS shared
FROM exports e
INNER JOIN active_storage_attachments a ON a.record_id = e.id AND a.record_type = 'ExportRecord' AND a.name = 'file'
INNER JOIN active_storage_blobs b ON b.id = a.blob_id
WHERE e.id = @export_id AND e.space_id = @space_id;

-- name: DeleteLegacyExportFiles :exec
-- Removes only this export's attachments and blobs with no remaining references.
-- The single statement keeps metadata recoverable if deletion fails.
--
-- [Ja] このエクスポートの関連と、参照が残らない blob だけを削除する。
-- 単一の文にまとめ、削除失敗時にメタデータから再試行できるようにする。
WITH removed AS (
    DELETE FROM active_storage_attachments a USING exports e
    WHERE e.id = @export_id AND e.space_id = @space_id
      AND a.record_type = 'ExportRecord' AND a.record_id = e.id AND a.name = 'file'
    RETURNING a.id, a.blob_id
)
DELETE FROM active_storage_blobs b
WHERE b.id IN (SELECT blob_id FROM removed)
  AND NOT EXISTS (
      SELECT 1 FROM active_storage_attachments a
      WHERE a.blob_id = b.id AND a.id NOT IN (SELECT id FROM removed)
  );
