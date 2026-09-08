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
-- [Ja] 期待する状態のいずれかにある場合に限り、エクスポートを指定の状態へ進める。停止したと
-- 判断された後で生き返ったワーカーが、自分を置き換えたエクスポートの状態を上書きしないよう、
-- 遷移は単なる書き込みではなく条件付きにする。現在の状態が期待する状態のいずれでもない場合は
-- 行を返さない。
--
-- heartbeat_at と object_key はパラメータが NULL のとき現在の値を保つ。これらについて言うことの
-- 無い遷移が、値を消してしまわないようにするためである。
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

-- name: DeleteExportStatusesByExport :exec
-- Deletes the Rails-era status history of the export. The rows reference exports without
-- ON DELETE CASCADE, so they have to go before the export itself. The statement disappears
-- with the export_statuses table once the Rails export code is removed.
--
-- [Ja] Rails 時代のエクスポートの状態履歴を削除する。これらの行は ON DELETE CASCADE 無しで
-- exports を参照しているため、エクスポート本体より先に消す必要がある。この文は Rails 版の
-- エクスポートのコードを削除するときに export_statuses テーブルごと無くなる。
DELETE FROM export_statuses WHERE export_id = @export_id AND space_id = @space_id;

-- name: DeleteExport :exec
-- Deletes the export record. The ZIP object it points at is deleted by the caller.
--
-- [Ja] エクスポートのレコードを削除する。参照している ZIP オブジェクトの削除は呼び出し側が行う。
DELETE FROM exports WHERE id = @id AND space_id = @space_id;
