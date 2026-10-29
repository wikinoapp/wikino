-- migrate:up
SET LOCAL lock_timeout = '5s';

-- トークンの宛先。0は公開API、1はMCPサーバー。認可コードに記録し、交換で発行するトークンに
-- 引き継ぐ。宛先の違うトークンを取り違えて受け付けないよう、照合で確かめる。
-- 既存の行は公開APIの認可で作ったもので、既定値の0になる。
ALTER TABLE oauth_authorization_codes
    ADD COLUMN audience INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT chk_oauth_authorization_codes_audience CHECK (audience IN (0, 1));
ALTER TABLE oauth_access_tokens
    ADD COLUMN audience INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT chk_oauth_access_tokens_audience CHECK (audience IN (0, 1));
ALTER TABLE oauth_refresh_tokens
    ADD COLUMN audience INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT chk_oauth_refresh_tokens_audience CHECK (audience IN (0, 1));

-- migrate:down
SET LOCAL lock_timeout = '5s';

ALTER TABLE oauth_refresh_tokens DROP COLUMN audience;
ALTER TABLE oauth_access_tokens DROP COLUMN audience;
ALTER TABLE oauth_authorization_codes DROP COLUMN audience;
