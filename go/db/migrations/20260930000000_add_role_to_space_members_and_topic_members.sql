-- migrate:up
-- 認可はスコープで継続し、ロールの保存を先に導入する。
-- 旧版の書き込みを受け入れるため、ロールはNULL可で追加する。
-- 既存行のバックフィルは、ロールを書かない旧版の停止後に別リリースで行う。
ALTER TABLE space_members
    ADD COLUMN role TEXT,
    ADD CONSTRAINT chk_space_members_role CHECK (role IN ('admin', 'editor', 'viewer'));

-- NULLは、そのトピックで追加する権限が無いことを表す。
ALTER TABLE topic_members
    ADD COLUMN role TEXT,
    ADD CONSTRAINT chk_topic_members_role CHECK (role IN ('admin', 'editor', 'viewer'));

-- migrate:down
ALTER TABLE topic_members
    DROP COLUMN role;

ALTER TABLE space_members
    DROP COLUMN role;
