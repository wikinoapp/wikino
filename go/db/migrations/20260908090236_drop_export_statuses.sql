-- migrate:up

-- Drop the Rails-era status history of exports. The Go version keeps the state on the
-- exports row itself, and the Rails export code that wrote these rows is gone, so the
-- table has no writer and no reader left.
--
-- [Ja] Rails 時代のエクスポートの状態履歴を削除する。Go 版は状態を exports の行自体に
-- 持ち、これらの行を書いていた Rails 版のエクスポートのコードも無くなったため、この
-- テーブルには書き手も読み手も残っていない。
DROP TABLE export_statuses;

-- migrate:down

CREATE TABLE export_statuses (
    id uuid DEFAULT generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    export_id uuid NOT NULL,
    kind integer NOT NULL,
    changed_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);

ALTER TABLE ONLY export_statuses
    ADD CONSTRAINT export_statuses_pkey PRIMARY KEY (id);

CREATE INDEX index_export_statuses_on_export_id ON export_statuses USING btree (export_id);

CREATE INDEX index_export_statuses_on_space_id ON export_statuses USING btree (space_id);

ALTER TABLE ONLY export_statuses
    ADD CONSTRAINT fk_rails_a8d9f2050b
        FOREIGN KEY (export_id) REFERENCES exports(id) ON DELETE CASCADE;

ALTER TABLE ONLY export_statuses
    ADD CONSTRAINT fk_rails_cab71249f9
        FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE;
