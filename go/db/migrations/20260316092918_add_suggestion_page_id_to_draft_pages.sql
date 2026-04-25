-- migrate:up

ALTER TABLE draft_pages
    ADD COLUMN suggestion_page_id UUID REFERENCES suggestion_pages(id);

CREATE INDEX index_draft_pages_on_suggestion_page_id ON draft_pages (suggestion_page_id) WHERE suggestion_page_id IS NOT NULL;

-- migrate:down

DROP INDEX IF EXISTS index_draft_pages_on_suggestion_page_id;

ALTER TABLE draft_pages
    DROP COLUMN suggestion_page_id;
