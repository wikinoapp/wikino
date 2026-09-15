-- migrate:up

-- Rails側 (Goで追加されたこれらのFKを知らない) が親を削除しても
-- 外部キー違反で失敗しないよう、suggestions系テーブルとdraft_pagesの
-- Go追加カラムのFKをON DELETEを明示して作り直す。純粋な従属データは
-- CASCADEし、nullableな参照 (注目画像・ページリビジョン・下書きから参照
-- する提案ページ) は参照元の行を残すためSET NULLにする。
ALTER TABLE suggestions
    DROP CONSTRAINT suggestions_space_id_fkey,
    ADD CONSTRAINT suggestions_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    DROP CONSTRAINT suggestions_topic_id_fkey,
    ADD CONSTRAINT suggestions_topic_id_fkey
        FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    DROP CONSTRAINT suggestions_created_space_member_id_fkey,
    ADD CONSTRAINT suggestions_created_space_member_id_fkey
        FOREIGN KEY (created_space_member_id) REFERENCES space_members(id) ON DELETE CASCADE;

ALTER TABLE suggestion_pages
    DROP CONSTRAINT suggestion_pages_space_id_fkey,
    ADD CONSTRAINT suggestion_pages_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    DROP CONSTRAINT suggestion_pages_suggestion_id_fkey,
    ADD CONSTRAINT suggestion_pages_suggestion_id_fkey
        FOREIGN KEY (suggestion_id) REFERENCES suggestions(id) ON DELETE CASCADE,
    DROP CONSTRAINT suggestion_pages_page_id_fkey,
    ADD CONSTRAINT suggestion_pages_page_id_fkey
        FOREIGN KEY (page_id) REFERENCES pages(id) ON DELETE CASCADE,
    DROP CONSTRAINT suggestion_pages_page_revision_id_fkey,
    ADD CONSTRAINT suggestion_pages_page_revision_id_fkey
        FOREIGN KEY (page_revision_id) REFERENCES page_revisions(id) ON DELETE SET NULL,
    DROP CONSTRAINT fk_suggestion_pages_featured_image_attachment_id,
    ADD CONSTRAINT fk_suggestion_pages_featured_image_attachment_id
        FOREIGN KEY (featured_image_attachment_id) REFERENCES attachments(id) ON DELETE SET NULL;

ALTER TABLE suggestion_page_revisions
    DROP CONSTRAINT suggestion_page_revisions_space_id_fkey,
    ADD CONSTRAINT suggestion_page_revisions_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    DROP CONSTRAINT suggestion_page_revisions_suggestion_page_id_fkey,
    ADD CONSTRAINT suggestion_page_revisions_suggestion_page_id_fkey
        FOREIGN KEY (suggestion_page_id) REFERENCES suggestion_pages(id) ON DELETE CASCADE,
    DROP CONSTRAINT suggestion_page_revisions_editor_space_member_id_fkey,
    ADD CONSTRAINT suggestion_page_revisions_editor_space_member_id_fkey
        FOREIGN KEY (editor_space_member_id) REFERENCES space_members(id) ON DELETE CASCADE;

ALTER TABLE suggestion_comments
    DROP CONSTRAINT suggestion_comments_space_id_fkey,
    ADD CONSTRAINT suggestion_comments_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    DROP CONSTRAINT suggestion_comments_suggestion_id_fkey,
    ADD CONSTRAINT suggestion_comments_suggestion_id_fkey
        FOREIGN KEY (suggestion_id) REFERENCES suggestions(id) ON DELETE CASCADE,
    DROP CONSTRAINT suggestion_comments_created_space_member_id_fkey,
    ADD CONSTRAINT suggestion_comments_created_space_member_id_fkey
        FOREIGN KEY (created_space_member_id) REFERENCES space_members(id) ON DELETE CASCADE;

ALTER TABLE draft_pages
    DROP CONSTRAINT draft_pages_suggestion_page_id_fkey,
    ADD CONSTRAINT draft_pages_suggestion_page_id_fkey
        FOREIGN KEY (suggestion_page_id) REFERENCES suggestion_pages(id) ON DELETE SET NULL,
    DROP CONSTRAINT fk_draft_pages_featured_image_attachment_id,
    ADD CONSTRAINT fk_draft_pages_featured_image_attachment_id
        FOREIGN KEY (featured_image_attachment_id) REFERENCES attachments(id) ON DELETE SET NULL;

-- インデックスの無いFKカラムにインデックスを張り、親の行の削除時に
-- 子テーブルがシーケンシャルスキャンされないようにする。他のFKカラムは
-- 既存インデックス (単一カラムまたは複合インデックスの先頭カラム) で対応
-- 済み。nullableなカラムは大半の行がNULLのため、
-- index_draft_pages_on_suggestion_page_idに合わせて部分インデックスにする。
CREATE INDEX idx_suggestion_pages_page_id ON suggestion_pages(page_id);
CREATE INDEX idx_suggestion_pages_page_revision_id ON suggestion_pages(page_revision_id) WHERE page_revision_id IS NOT NULL;
CREATE INDEX idx_suggestion_pages_featured_image_attachment_id ON suggestion_pages(featured_image_attachment_id) WHERE featured_image_attachment_id IS NOT NULL;
CREATE INDEX idx_suggestion_page_revisions_editor_space_member_id ON suggestion_page_revisions(editor_space_member_id);
CREATE INDEX idx_suggestion_comments_created_space_member_id ON suggestion_comments(created_space_member_id);
CREATE INDEX index_draft_pages_on_featured_image_attachment_id ON draft_pages(featured_image_attachment_id) WHERE featured_image_attachment_id IS NOT NULL;

-- migrate:down

DROP INDEX IF EXISTS index_draft_pages_on_featured_image_attachment_id;
DROP INDEX IF EXISTS idx_suggestion_comments_created_space_member_id;
DROP INDEX IF EXISTS idx_suggestion_page_revisions_editor_space_member_id;
DROP INDEX IF EXISTS idx_suggestion_pages_featured_image_attachment_id;
DROP INDEX IF EXISTS idx_suggestion_pages_page_revision_id;
DROP INDEX IF EXISTS idx_suggestion_pages_page_id;

ALTER TABLE draft_pages
    DROP CONSTRAINT draft_pages_suggestion_page_id_fkey,
    ADD CONSTRAINT draft_pages_suggestion_page_id_fkey
        FOREIGN KEY (suggestion_page_id) REFERENCES suggestion_pages(id),
    DROP CONSTRAINT fk_draft_pages_featured_image_attachment_id,
    ADD CONSTRAINT fk_draft_pages_featured_image_attachment_id
        FOREIGN KEY (featured_image_attachment_id) REFERENCES attachments(id);

ALTER TABLE suggestion_comments
    DROP CONSTRAINT suggestion_comments_space_id_fkey,
    ADD CONSTRAINT suggestion_comments_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id),
    DROP CONSTRAINT suggestion_comments_suggestion_id_fkey,
    ADD CONSTRAINT suggestion_comments_suggestion_id_fkey
        FOREIGN KEY (suggestion_id) REFERENCES suggestions(id),
    DROP CONSTRAINT suggestion_comments_created_space_member_id_fkey,
    ADD CONSTRAINT suggestion_comments_created_space_member_id_fkey
        FOREIGN KEY (created_space_member_id) REFERENCES space_members(id);

ALTER TABLE suggestion_page_revisions
    DROP CONSTRAINT suggestion_page_revisions_space_id_fkey,
    ADD CONSTRAINT suggestion_page_revisions_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id),
    DROP CONSTRAINT suggestion_page_revisions_suggestion_page_id_fkey,
    ADD CONSTRAINT suggestion_page_revisions_suggestion_page_id_fkey
        FOREIGN KEY (suggestion_page_id) REFERENCES suggestion_pages(id),
    DROP CONSTRAINT suggestion_page_revisions_editor_space_member_id_fkey,
    ADD CONSTRAINT suggestion_page_revisions_editor_space_member_id_fkey
        FOREIGN KEY (editor_space_member_id) REFERENCES space_members(id);

ALTER TABLE suggestion_pages
    DROP CONSTRAINT suggestion_pages_space_id_fkey,
    ADD CONSTRAINT suggestion_pages_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id),
    DROP CONSTRAINT suggestion_pages_suggestion_id_fkey,
    ADD CONSTRAINT suggestion_pages_suggestion_id_fkey
        FOREIGN KEY (suggestion_id) REFERENCES suggestions(id),
    DROP CONSTRAINT suggestion_pages_page_id_fkey,
    ADD CONSTRAINT suggestion_pages_page_id_fkey
        FOREIGN KEY (page_id) REFERENCES pages(id),
    DROP CONSTRAINT suggestion_pages_page_revision_id_fkey,
    ADD CONSTRAINT suggestion_pages_page_revision_id_fkey
        FOREIGN KEY (page_revision_id) REFERENCES page_revisions(id),
    DROP CONSTRAINT fk_suggestion_pages_featured_image_attachment_id,
    ADD CONSTRAINT fk_suggestion_pages_featured_image_attachment_id
        FOREIGN KEY (featured_image_attachment_id) REFERENCES attachments(id);

ALTER TABLE suggestions
    DROP CONSTRAINT suggestions_space_id_fkey,
    ADD CONSTRAINT suggestions_space_id_fkey
        FOREIGN KEY (space_id) REFERENCES spaces(id),
    DROP CONSTRAINT suggestions_topic_id_fkey,
    ADD CONSTRAINT suggestions_topic_id_fkey
        FOREIGN KEY (topic_id) REFERENCES topics(id),
    DROP CONSTRAINT suggestions_created_space_member_id_fkey,
    ADD CONSTRAINT suggestions_created_space_member_id_fkey
        FOREIGN KEY (created_space_member_id) REFERENCES space_members(id);
