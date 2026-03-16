-- migrate:up

CREATE TABLE suggestion_page_revisions (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id),
    suggestion_page_id UUID NOT NULL REFERENCES suggestion_pages(id),
    editor_space_member_id UUID NOT NULL REFERENCES space_members(id),
    title VARCHAR,
    body VARCHAR NOT NULL DEFAULT '',
    body_html VARCHAR NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_suggestion_page_revisions_suggestion_page_id_created_at ON suggestion_page_revisions(suggestion_page_id, created_at);
CREATE INDEX idx_suggestion_page_revisions_space_id ON suggestion_page_revisions(space_id);

-- migrate:down

DROP INDEX IF EXISTS idx_suggestion_page_revisions_space_id;
DROP INDEX IF EXISTS idx_suggestion_page_revisions_suggestion_page_id_created_at;
DROP TABLE IF EXISTS suggestion_page_revisions;
