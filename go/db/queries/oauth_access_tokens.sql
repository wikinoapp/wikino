-- name: CreateOAuthAccessToken :one
-- OAuthのアクセストークンを作成する
INSERT INTO oauth_access_tokens (
    oauth_grant_id,
    space_id,
    token_digest,
    scopes,
    expires_at,
    created_at,
    updated_at
) VALUES (
    @oauth_grant_id,
    @space_id,
    @token_digest,
    @scopes::text[],
    @expires_at,
    @now,
    @now
) RETURNING *;

-- name: FindOAuthAccessTokenByTokenDigest :one
-- トークンダイジェストでOAuthのアクセストークンを取得する。
-- space_idを条件に含めない例外のクエリである。トークン認証はスペースを特定する前の入口で、
-- スペースはこのトークンから決まるためである。呼び出し側は、ここで得たspace_idを以降の
-- クエリの条件に必ず含める。
SELECT * FROM oauth_access_tokens WHERE token_digest = @token_digest;

-- name: RevokeOAuthAccessTokensByOAuthGrant :exec
-- 許可に属する失効していないアクセストークンをすべて失効する
UPDATE oauth_access_tokens
SET revoked_at = @now::timestamptz, updated_at = @now
WHERE oauth_grant_id = @oauth_grant_id
  AND space_id = @space_id
  AND revoked_at IS NULL;

-- name: RevokeOAuthAccessToken :exec
-- アクセストークンを1つ失効する (RFC 7009)。既に失効しているものは変えない
UPDATE oauth_access_tokens
SET revoked_at = @now::timestamptz, updated_at = @now
WHERE id = @id
  AND space_id = @space_id
  AND revoked_at IS NULL;
