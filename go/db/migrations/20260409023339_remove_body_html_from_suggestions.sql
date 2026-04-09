-- migrate:up

ALTER TABLE suggestions DROP COLUMN body_html;

-- migrate:down

ALTER TABLE suggestions ADD COLUMN body_html VARCHAR NOT NULL DEFAULT '';
