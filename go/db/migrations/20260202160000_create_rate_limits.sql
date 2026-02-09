-- migrate:up

-- rate_limitsテーブル: Rate Limiting用のカウンターを保存する
CREATE TABLE rate_limits (
    id VARCHAR NOT NULL PRIMARY KEY DEFAULT generate_ulid(),
    key VARCHAR NOT NULL,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(key, window_start)
);

-- keyとwindow_startの複合インデックス（クエリの高速化）
CREATE INDEX idx_rate_limits_key_window_start ON rate_limits(key, window_start);

-- 古いレコード削除用のインデックス
CREATE INDEX idx_rate_limits_window_start ON rate_limits(window_start);

-- migrate:down

DROP INDEX IF EXISTS idx_rate_limits_window_start;
DROP INDEX IF EXISTS idx_rate_limits_key_window_start;
DROP TABLE IF EXISTS rate_limits;
