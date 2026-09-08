-- migrate:up

-- Move the export state onto the exports row itself: the status, when it last
-- changed, the heartbeat the worker updates while it runs, and the key of the
-- ZIP object in the bucket. The Rails version keeps the state as
-- export_statuses history rows and the ZIP as an ActiveStorage attachment, but
-- no screen reads the history, and reproducing the ActiveStorage record
-- conventions from Go would be fragile.
--
-- status_changed_at carries a now() default so that it can be NOT NULL for the
-- rows the backfill below does not reach. heartbeat_at and object_key stay
-- nullable: an export has neither until the worker starts and finishes.
--
-- [Ja] エクスポートの状態を exports の行そのものへ移す。状態・最後に状態が変わった
-- 時刻・ワーカーが処理中に更新する heartbeat・バケット内の ZIP オブジェクトのキーを
-- 持たせる。Rails 版は状態を export_statuses の履歴行として、ZIP を ActiveStorage の
-- attachment として持っているが、履歴を読む画面は無く、ActiveStorage のレコードの
-- 規約を Go から再現するのは壊れやすい。
--
-- status_changed_at に now() の既定値を付けるのは、下の backfill が届かない行に対しても
-- NOT NULL にできるようにするため。heartbeat_at と object_key は、ワーカーが開始し完了
-- するまで値を持たないので nullable のままにする。
ALTER TABLE exports ADD COLUMN status INTEGER NOT NULL DEFAULT 0;
ALTER TABLE exports ADD COLUMN status_changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now();
ALTER TABLE exports ADD COLUMN heartbeat_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE exports ADD COLUMN object_key VARCHAR;

-- Fill the existing rows from the newest export_statuses row of each export.
-- The kind values (queued / started / succeeded / failed) are the same integers
-- on both sides, so they carry over as they are. changed_at is a timestamp
-- without time zone that Rails writes in UTC, hence the explicit
-- AT TIME ZONE 'UTC'.
--
-- object_key is left NULL: the ZIP of an existing export lives under an
-- ActiveStorage blob key, and those exports expire within a day of finishing.
--
-- [Ja] 既存行を、各エクスポートの最新の export_statuses 行から埋める。kind の値
-- (queued / started / succeeded / failed) は両者で同じ整数なので、そのまま持ち越せる。
-- changed_at は Rails が UTC で書いている timestamp without time zone なので、
-- AT TIME ZONE 'UTC' を明示する。
--
-- object_key は NULL のままにする。既存のエクスポートの ZIP は ActiveStorage の blob の
-- キーの下にあり、それらのエクスポートは完了から 1 日で期限切れになるためである。
UPDATE exports e
SET status = s.kind,
    status_changed_at = s.changed_at AT TIME ZONE 'UTC'
FROM (
    SELECT DISTINCT ON (export_id) export_id, kind, changed_at
    FROM export_statuses
    ORDER BY export_id, changed_at DESC
) s
WHERE e.id = s.export_id;

-- Give up on the exports that never finished. The Rails version's workers stop with the deploy
-- that hands the export over to Go, so nothing will ever advance a row left in queued or
-- started: the screen would keep showing it as in progress, and the space could never start
-- another export. The rows the backfill above did not reach (an export with no export_statuses
-- row) are queued by the column default and are covered by the same condition.
--
-- [Ja] 完了しなかったエクスポートを諦めさせる。Rails 版のワーカーは、エクスポートを Go 版へ
-- 引き渡すデプロイで止まるため、queued や started のまま残った行を進めるものが居なくなる。
-- そのままでは画面が処理中を表示し続け、そのスペースで新しいエクスポートを開始できなくなる。
-- 上の backfill が届かなかった行 (export_statuses を 1 行も持たないエクスポート) も、列の
-- 既定値により queued なので同じ条件で拾える。
UPDATE exports
SET status = 3, status_changed_at = now()
WHERE status IN (0, 1);

-- migrate:down

ALTER TABLE exports DROP COLUMN object_key;
ALTER TABLE exports DROP COLUMN heartbeat_at;
ALTER TABLE exports DROP COLUMN status_changed_at;
ALTER TABLE exports DROP COLUMN status;
