-- migrate:up

-- Rails側 (これらのテーブルを知らなくなった) が親を削除しても外部キー違反で
-- 失敗しないよう、FKをON DELETE CASCADE付きで作り直す。スペースの削除は
-- 引き続きRails側で動く一方、exportsとその状態はRails側のエクスポートの
-- コードを削除した時点でGoが持つ行になった。ここに挙げたFKのカラムはいずれも
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
