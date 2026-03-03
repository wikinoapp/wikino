-- migrate:up

CREATE TABLE feature_flags (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_feature_flags_user_id ON feature_flags(user_id);
CREATE INDEX idx_feature_flags_name ON feature_flags(name);

-- migrate:down

DROP INDEX IF EXISTS idx_feature_flags_name;
DROP INDEX IF EXISTS idx_feature_flags_user_id;
DROP TABLE IF EXISTS feature_flags;
