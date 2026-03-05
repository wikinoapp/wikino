-- migrate:up

CREATE TABLE draft_page_revisions (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    draft_page_id UUID NOT NULL REFERENCES draft_pages(id),
    space_member_id UUID NOT NULL REFERENCES space_members(id),
    title VARCHAR NOT NULL,
    body VARCHAR NOT NULL,
    body_html VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_draft_page_revisions_draft_page_id_created_at ON draft_page_revisions(draft_page_id, created_at);

-- migrate:down

DROP INDEX IF EXISTS idx_draft_page_revisions_draft_page_id_created_at;
DROP TABLE IF EXISTS draft_page_revisions;
