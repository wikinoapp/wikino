-- migrate:up

-- エクスポートの状態をexportsの行そのものへ移す。状態・最後に状態が変わった
-- 時刻・ワーカーが処理中に更新するheartbeat・バケット内のZIPオブジェクトのキーを
-- 持たせる。Rails版は状態をexport_statusesの履歴行として、ZIPをActiveStorageの
-- attachmentとして持っているが、履歴を読む画面は無く、ActiveStorageのレコードの
-- 規約をGoから再現するのは壊れやすい。
--
-- status_changed_atにnow() の既定値を付けるのは、下のbackfillが届かない行に対しても
-- NOT NULLにできるようにするため。heartbeat_atとobject_keyは、ワーカーが開始し完了
-- するまで値を持たないのでnullableのままにする。
ALTER TABLE exports ADD COLUMN status INTEGER NOT NULL DEFAULT 0;
ALTER TABLE exports ADD COLUMN status_changed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now();
ALTER TABLE exports ADD COLUMN heartbeat_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE exports ADD COLUMN object_key VARCHAR;

-- 既存行を、各エクスポートの最新のexport_statuses行から埋める。kindの値
-- (queued / started / succeeded / failed) は両者で同じ整数なので、そのまま持ち越せる。
-- changed_atはRailsがUTCで書いているtimestamp without time zoneなので、
-- AT TIME ZONE 'UTC' を明示する。
--
-- object_keyはNULLのままにする。既存のエクスポートのZIPはActiveStorageのblobの
-- キーの下にあり、それらのエクスポートは完了から1日で期限切れになるためである。
UPDATE exports e
SET status = s.kind,
    status_changed_at = s.changed_at AT TIME ZONE 'UTC'
FROM (
    SELECT DISTINCT ON (export_id) export_id, kind, changed_at
    FROM export_statuses
    ORDER BY export_id, changed_at DESC
) s
WHERE e.id = s.export_id;

-- 完了しなかったエクスポートを諦めさせる。Rails版のワーカーは、エクスポートをGo版へ
-- 引き渡すデプロイで止まるため、queuedやstartedのまま残った行を進めるものが居なくなる。
-- そのままでは画面が処理中を表示し続け、そのスペースで新しいエクスポートを開始できなくなる。
-- 上のbackfillが届かなかった行 (export_statusesを1行も持たないエクスポート) も、列の
-- 既定値によりqueuedなので同じ条件で拾える。
UPDATE exports
SET status = 3, status_changed_at = now()
WHERE status IN (0, 1);

-- migrate:down

ALTER TABLE exports DROP COLUMN object_key;
ALTER TABLE exports DROP COLUMN heartbeat_at;
ALTER TABLE exports DROP COLUMN status_changed_at;
ALTER TABLE exports DROP COLUMN status;
