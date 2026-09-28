-- name: CreateOAuthGrant :one
-- OAuthアプリの許可を作成する
INSERT INTO oauth_grants (
    oauth_application_id,
    space_id,
    space_member_id,
    scopes,
    created_at,
    updated_at
) VALUES (
    @oauth_application_id,
    @space_id,
    @space_member_id,
    @scopes::text[],
    @now,
    @now
) RETURNING *;

-- name: FindOAuthGrantByID :one
-- IDでOAuthアプリの許可を取得する
SELECT * FROM oauth_grants
WHERE id = @id
  AND space_id = @space_id;

-- name: FindUnrevokedOAuthGrantByApplicationAndSpaceMember :one
-- スペースメンバーがOAuthアプリに与えている、失効していない許可を取得する
SELECT * FROM oauth_grants
WHERE oauth_application_id = @oauth_application_id
  AND space_id = @space_id
  AND space_member_id = @space_member_id
  AND revoked_at IS NULL;

-- name: ListUnrevokedOAuthGrantsBySpaceMember :many
-- スペースメンバーが与えている失効していない許可を、新しい順に取得する。削除したアプリの
-- 許可は削除のときに失効しているため、ここには出ない
SELECT * FROM oauth_grants
WHERE space_id = @space_id
  AND space_member_id = @space_member_id
  AND revoked_at IS NULL
ORDER BY created_at DESC, id DESC;

-- name: RevokeOAuthGrant :one
-- スペースメンバー自身のOAuthアプリの許可と、許可に属するアクセストークン・リフレッシュトークンを
-- すべて失効する。1文で行うため、許可だけが失効してトークンが使える状態は途中にも残らない。
-- 対象が無いか既に失効している場合は何も変えずに行を返さない
WITH revoked_grant AS (
    UPDATE oauth_grants
    SET revoked_at = @now::timestamptz, updated_at = @now
    WHERE oauth_grants.id = @id
      AND oauth_grants.space_id = @space_id
      AND oauth_grants.space_member_id = @space_member_id
      AND oauth_grants.revoked_at IS NULL
    RETURNING oauth_grants.*
),
revoked_access_tokens AS (
    UPDATE oauth_access_tokens
    SET revoked_at = @now::timestamptz, updated_at = @now
    WHERE oauth_access_tokens.oauth_grant_id IN (SELECT revoked_grant.id FROM revoked_grant)
      AND oauth_access_tokens.space_id = @space_id
      AND oauth_access_tokens.revoked_at IS NULL
),
revoked_refresh_tokens AS (
    UPDATE oauth_refresh_tokens
    SET revoked_at = @now::timestamptz, updated_at = @now
    WHERE oauth_refresh_tokens.oauth_grant_id IN (SELECT revoked_grant.id FROM revoked_grant)
      AND oauth_refresh_tokens.space_id = @space_id
      AND oauth_refresh_tokens.revoked_at IS NULL
)
SELECT * FROM revoked_grant;
