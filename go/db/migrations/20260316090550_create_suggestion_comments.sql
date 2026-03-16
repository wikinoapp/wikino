-- migrate:up

CREATE TABLE suggestion_comments (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id),
    suggestion_id UUID NOT NULL REFERENCES suggestions(id),
    created_space_member_id UUID NOT NULL REFERENCES space_members(id),
    body VARCHAR NOT NULL DEFAULT '',
    body_html VARCHAR NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_suggestion_comments_suggestion_id_created_at ON suggestion_comments(suggestion_id, created_at);
CREATE INDEX idx_suggestion_comments_space_id ON suggestion_comments(space_id);

-- migrate:down

DROP INDEX IF EXISTS idx_suggestion_comments_space_id;
DROP INDEX IF EXISTS idx_suggestion_comments_suggestion_id_created_at;
DROP TABLE IF EXISTS suggestion_comments;
