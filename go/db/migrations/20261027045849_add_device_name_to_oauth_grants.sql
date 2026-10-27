-- migrate:up
SET LOCAL lock_timeout = '5s';

-- 連携の一覧で端末を見分けるための、クライアントが申告した端末名。NULLは端末名が送られなかったことを
-- 表す。長さの上限はアプリケーションで確かめる。
-- デバイス認可では要求で受け取って保存し、利用者が許可したときに許可へ写す。
ALTER TABLE oauth_grants ADD COLUMN device_name VARCHAR;
ALTER TABLE oauth_device_authorizations ADD COLUMN device_name VARCHAR;

-- migrate:down
SET LOCAL lock_timeout = '5s';

ALTER TABLE oauth_device_authorizations DROP COLUMN device_name;
ALTER TABLE oauth_grants DROP COLUMN device_name;
