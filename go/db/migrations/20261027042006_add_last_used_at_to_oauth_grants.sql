-- migrate:up
SET LOCAL lock_timeout = '5s';

-- 連携の一覧で端末を見分けるための最終利用日時。トークンの発行とアクセストークンの照合で更新する。
-- NULLはまだトークンを発行していないことを表す。
ALTER TABLE oauth_grants ADD COLUMN last_used_at TIMESTAMP WITH TIME ZONE;

-- migrate:down
SET LOCAL lock_timeout = '5s';

ALTER TABLE oauth_grants DROP COLUMN last_used_at;
