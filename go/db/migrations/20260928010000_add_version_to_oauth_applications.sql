-- migrate:up
-- 編集フォームが読み込んだ版と保存時の版を比較し、古い内容による上書きを防ぐ。
ALTER TABLE oauth_applications
    ADD COLUMN version BIGINT NOT NULL DEFAULT 1;

-- migrate:down
ALTER TABLE oauth_applications
    DROP COLUMN version;
