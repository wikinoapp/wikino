-- migrate:up
SET LOCAL lock_timeout = '5s';

-- 許可は認可 (同意) ごとに作るため、同じアプリ・メンバーの失効していない許可が複数並ぶ。
-- 一意性を外し、同じアプリ・メンバーの許可を数える・選ぶための索引として残す。
DROP INDEX idx_oauth_grants_unrevoked_application_member;
CREATE INDEX idx_oauth_grants_unrevoked_application_member
    ON oauth_grants(oauth_application_id, space_member_id) WHERE revoked_at IS NULL;

-- migrate:down
SET LOCAL lock_timeout = '5s';

-- 同じアプリ・メンバーの失効していない許可が複数あると、一意の索引を作れず失敗する。
DROP INDEX idx_oauth_grants_unrevoked_application_member;
CREATE UNIQUE INDEX idx_oauth_grants_unrevoked_application_member
    ON oauth_grants(oauth_application_id, space_member_id) WHERE revoked_at IS NULL;
