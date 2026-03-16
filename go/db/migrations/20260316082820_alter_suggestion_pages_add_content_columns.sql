-- migrate:up

ALTER TABLE suggestion_pages
    ADD COLUMN title VARCHAR,
    ADD COLUMN body VARCHAR NOT NULL DEFAULT '',
    ADD COLUMN body_html VARCHAR NOT NULL DEFAULT '';

ALTER TABLE suggestion_pages
    ALTER COLUMN page_revision_id SET NOT NULL;

ALTER TABLE suggestion_pages
    DROP COLUMN latest_revision_id;

-- migrate:down

ALTER TABLE suggestion_pages
    ADD COLUMN latest_revision_id UUID;

ALTER TABLE suggestion_pages
    ALTER COLUMN page_revision_id DROP NOT NULL;

ALTER TABLE suggestion_pages
    DROP COLUMN body_html,
    DROP COLUMN body,
    DROP COLUMN title;
