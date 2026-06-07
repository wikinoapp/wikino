-- migrate:up

-- Recreate the FKs with ON DELETE CASCADE so that deleting parents from the
-- Rails side (which does not know about this Go-added table) cannot leave
-- orphaned revisions or fail with a foreign key violation.
--
-- [Ja] Rails 側 (Go で追加されたこのテーブルを知らない) が親を削除しても、
-- リビジョンが孤児として残ったり外部キー違反で失敗したりしないよう、
-- FK を ON DELETE CASCADE 付きで作り直す。
ALTER TABLE draft_page_revisions
    DROP CONSTRAINT draft_page_revisions_draft_page_id_fkey,
    ADD CONSTRAINT draft_page_revisions_draft_page_id_fkey
        FOREIGN KEY (draft_page_id) REFERENCES draft_pages(id) ON DELETE CASCADE,
    DROP CONSTRAINT draft_page_revisions_space_id_fkey,
    ADD CONSTRAINT draft_page_revisions_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    DROP CONSTRAINT draft_page_revisions_space_member_id_fkey,
    ADD CONSTRAINT draft_page_revisions_space_member_id_fkey
        FOREIGN KEY (space_member_id) REFERENCES space_members(id) ON DELETE CASCADE;

-- Index the cascading FK columns so that deleting a parent row does not
-- sequentially scan draft_page_revisions. draft_page_id is already covered
-- by idx_draft_page_revisions_draft_page_id_created_at (leading column).
--
-- [Ja] 親の行の削除時に draft_page_revisions がシーケンシャルスキャンされない
-- よう、CASCADE 対象の FK カラムにインデックスを張る。draft_page_id は
-- idx_draft_page_revisions_draft_page_id_created_at (先頭カラム) で対応済み。
CREATE INDEX idx_draft_page_revisions_space_id ON draft_page_revisions(space_id);
CREATE INDEX idx_draft_page_revisions_space_member_id ON draft_page_revisions(space_member_id);

-- migrate:down

DROP INDEX IF EXISTS idx_draft_page_revisions_space_member_id;
DROP INDEX IF EXISTS idx_draft_page_revisions_space_id;

ALTER TABLE draft_page_revisions
    DROP CONSTRAINT draft_page_revisions_draft_page_id_fkey,
    ADD CONSTRAINT draft_page_revisions_draft_page_id_fkey
        FOREIGN KEY (draft_page_id) REFERENCES draft_pages(id),
    DROP CONSTRAINT draft_page_revisions_space_id_fkey,
    ADD CONSTRAINT draft_page_revisions_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id),
    DROP CONSTRAINT draft_page_revisions_space_member_id_fkey,
    ADD CONSTRAINT draft_page_revisions_space_member_id_fkey
        FOREIGN KEY (space_member_id) REFERENCES space_members(id);
