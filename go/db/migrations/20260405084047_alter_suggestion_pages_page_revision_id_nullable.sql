-- migrate:up
ALTER TABLE suggestion_pages ALTER COLUMN page_revision_id DROP NOT NULL;

-- migrate:down
ALTER TABLE suggestion_pages ALTER COLUMN page_revision_id SET NOT NULL;
