-- migrate:up

ALTER TABLE space_members ADD COLUMN scopes text[] NOT NULL DEFAULT '{}';
ALTER TABLE topic_members ADD COLUMN scopes text[] NOT NULL DEFAULT '{}';

UPDATE space_members SET scopes = ARRAY['space:admin'];

-- migrate:down

ALTER TABLE topic_members DROP COLUMN scopes;
ALTER TABLE space_members DROP COLUMN scopes;
