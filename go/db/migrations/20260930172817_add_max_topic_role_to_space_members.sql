-- migrate:up
-- トピックで付けたロールの上限。NULLは制限しないことを表す。
-- 上限にトピック管理者を指定しても制限にならないため、値はトピック編集者とトピック閲覧者に限る。
SET LOCAL lock_timeout = '5s';

ALTER TABLE space_members
    ADD COLUMN max_topic_role TEXT,
    ADD CONSTRAINT chk_space_members_max_topic_role CHECK (max_topic_role IN ('editor', 'viewer'));

-- migrate:down
SET LOCAL lock_timeout = '5s';

ALTER TABLE space_members
    DROP COLUMN max_topic_role;
