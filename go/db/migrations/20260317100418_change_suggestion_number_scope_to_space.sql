-- migrate:up

DROP INDEX IF EXISTS idx_suggestions_topic_id_number;

CREATE UNIQUE INDEX idx_suggestions_space_id_number ON suggestions(space_id, number);

-- migrate:down

DROP INDEX IF EXISTS idx_suggestions_space_id_number;

CREATE UNIQUE INDEX idx_suggestions_topic_id_number ON suggestions(topic_id, number);
