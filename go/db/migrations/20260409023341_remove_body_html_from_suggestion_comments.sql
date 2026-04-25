-- migrate:up

ALTER TABLE suggestion_comments DROP COLUMN body_html;

-- migrate:down

ALTER TABLE suggestion_comments ADD COLUMN body_html VARCHAR NOT NULL DEFAULT '';
