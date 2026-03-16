-- migrate:up

ALTER TABLE suggestions ADD COLUMN number INTEGER NOT NULL DEFAULT 0;

CREATE UNIQUE INDEX idx_suggestions_topic_id_number ON suggestions(topic_id, number);

-- migrate:down

DROP INDEX IF EXISTS idx_suggestions_topic_id_number;

ALTER TABLE suggestions DROP COLUMN number;
