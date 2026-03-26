-- migrate:up

ALTER TABLE suggestion_comments ADD COLUMN number INTEGER NOT NULL DEFAULT 0;

UPDATE suggestion_comments
SET number = sub.rn
FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY suggestion_id ORDER BY created_at) AS rn
    FROM suggestion_comments
) sub
WHERE suggestion_comments.id = sub.id;

ALTER TABLE suggestion_comments ALTER COLUMN number DROP DEFAULT;

CREATE UNIQUE INDEX idx_suggestion_comments_suggestion_id_number ON suggestion_comments(suggestion_id, number);

-- migrate:down

DROP INDEX IF EXISTS idx_suggestion_comments_suggestion_id_number;

ALTER TABLE suggestion_comments DROP COLUMN number;
