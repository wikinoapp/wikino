-- migrate:up

-- Rails側 (Goで追加されたこのテーブルを知らない) が親を削除しても、
-- リビジョンが孤児として残ったり外部キー違反で失敗したりしないよう、
-- FKをON DELETE CASCADE付きで作り直す。
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

-- 親の行の削除時にdraft_page_revisionsがシーケンシャルスキャンされない
-- よう、CASCADE対象のFKカラムにインデックスを張る。draft_page_idは
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
