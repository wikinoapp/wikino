-- migrate:up

CREATE TABLE suggestion_pages (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id),
    suggestion_id UUID NOT NULL REFERENCES suggestions(id),
    page_id UUID NOT NULL REFERENCES pages(id),
    page_revision_id UUID REFERENCES page_revisions(id),
    latest_revision_id UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_suggestion_pages_suggestion_id_page_id ON suggestion_pages(suggestion_id, page_id);
CREATE INDEX idx_suggestion_pages_space_id ON suggestion_pages(space_id);

-- migrate:down

DROP INDEX IF EXISTS idx_suggestion_pages_space_id;
DROP INDEX IF EXISTS idx_suggestion_pages_suggestion_id_page_id;
DROP TABLE IF EXISTS suggestion_pages;
