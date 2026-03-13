-- migrate:up

CREATE TABLE suggestions (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id),
    topic_id UUID NOT NULL REFERENCES topics(id),
    created_space_member_id UUID NOT NULL REFERENCES space_members(id),
    title VARCHAR NOT NULL,
    body VARCHAR NOT NULL,
    body_html VARCHAR NOT NULL,
    status INTEGER NOT NULL DEFAULT 0,
    applied_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_suggestions_topic_id_status ON suggestions(topic_id, status);
CREATE INDEX idx_suggestions_space_id ON suggestions(space_id);
CREATE INDEX idx_suggestions_created_space_member_id ON suggestions(created_space_member_id);

-- migrate:down

DROP INDEX IF EXISTS idx_suggestions_created_space_member_id;
DROP INDEX IF EXISTS idx_suggestions_space_id;
DROP INDEX IF EXISTS idx_suggestions_topic_id_status;
DROP TABLE IF EXISTS suggestions;
