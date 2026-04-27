-- migrate:up

ALTER TABLE space_members DROP COLUMN role;
ALTER TABLE topic_members DROP COLUMN role;

-- migrate:down

ALTER TABLE space_members ADD COLUMN role integer NOT NULL DEFAULT 0;
ALTER TABLE topic_members ADD COLUMN role integer NOT NULL DEFAULT 0;
