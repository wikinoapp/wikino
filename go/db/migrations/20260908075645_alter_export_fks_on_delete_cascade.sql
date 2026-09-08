-- migrate:up

-- Recreate the FKs with ON DELETE CASCADE so that deleting parents from the
-- Rails side (which no longer knows about these tables) cannot fail with a
-- foreign key violation. Deleting a space still runs on Rails, while exports
-- and their statuses became Go-owned rows once the Rails export code was
-- removed. Every FK column here is already indexed, so no index is added.
--
-- [Ja] Rails 側 (これらのテーブルを知らなくなった) が親を削除しても外部キー違反で
-- 失敗しないよう、FK を ON DELETE CASCADE 付きで作り直す。スペースの削除は
-- 引き続き Rails 側で動く一方、exports とその状態は Rails 側のエクスポートの
-- コードを削除した時点で Go が持つ行になった。ここに挙げた FK のカラムはいずれも
-- インデックス済みのため、インデックスは追加しない。
ALTER TABLE exports
    DROP CONSTRAINT fk_rails_7fa4a1a0c0,
    ADD CONSTRAINT fk_rails_7fa4a1a0c0
        FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    DROP CONSTRAINT fk_rails_703ee3dae6,
    ADD CONSTRAINT fk_rails_703ee3dae6
        FOREIGN KEY (queued_by_id) REFERENCES space_members(id) ON DELETE CASCADE;

ALTER TABLE export_statuses
    DROP CONSTRAINT fk_rails_cab71249f9,
    ADD CONSTRAINT fk_rails_cab71249f9
        FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    DROP CONSTRAINT fk_rails_a8d9f2050b,
    ADD CONSTRAINT fk_rails_a8d9f2050b
        FOREIGN KEY (export_id) REFERENCES exports(id) ON DELETE CASCADE;

-- migrate:down

ALTER TABLE export_statuses
    DROP CONSTRAINT fk_rails_cab71249f9,
    ADD CONSTRAINT fk_rails_cab71249f9
        FOREIGN KEY (space_id) REFERENCES spaces(id),
    DROP CONSTRAINT fk_rails_a8d9f2050b,
    ADD CONSTRAINT fk_rails_a8d9f2050b
        FOREIGN KEY (export_id) REFERENCES exports(id);

ALTER TABLE exports
    DROP CONSTRAINT fk_rails_7fa4a1a0c0,
    ADD CONSTRAINT fk_rails_7fa4a1a0c0
        FOREIGN KEY (space_id) REFERENCES spaces(id),
    DROP CONSTRAINT fk_rails_703ee3dae6,
    ADD CONSTRAINT fk_rails_703ee3dae6
        FOREIGN KEY (queued_by_id) REFERENCES space_members(id);
