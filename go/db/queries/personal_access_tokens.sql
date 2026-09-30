-- name: CreatePersonalAccessToken :one
-- 個人アクセストークンを作成する
INSERT INTO personal_access_tokens (
    space_id,
    space_member_id,
    name,
    token_digest,
    token_last_chars,
    scopes,
    expires_at,
    created_at,
    updated_at
) VALUES (
    @space_id,
    @space_member_id,
    @name,
    @token_digest,
    @token_last_chars,
    @scopes::text[],
    @expires_at,
    @now,
    @now
) RETURNING *;

-- name: FindPersonalAccessTokenByTokenDigest :one
-- トークンダイジェストで個人アクセストークンを取得する。
-- space_idを条件に含めない例外のクエリである。トークン認証はスペースを特定する前の入口で、
-- スペースはこのトークンから決まるためである。呼び出し側は、ここで得たspace_idを以降の
-- クエリの条件に必ず含める。
SELECT * FROM personal_access_tokens WHERE token_digest = @token_digest;

-- name: UpdatePersonalAccessTokenLastUsedAt :exec
-- 最終使用日時を更新する。リクエストのたびに書き込まないよう、最終使用日時が
-- stale_beforeより古い (または未使用の) ときだけ更新する。利用者による変更ではないため、
-- updated_atは更新しない。
UPDATE personal_access_tokens
SET last_used_at = @now::timestamptz
WHERE id = @id
  AND space_id = @space_id
  AND revoked_at IS NULL
  AND (last_used_at IS NULL OR last_used_at < @stale_before::timestamptz);

-- name: RevokePersonalAccessToken :one
-- メンバー自身の個人アクセストークンを失効する。対象が無いか既に失効している場合は
-- 行を返さない。
UPDATE personal_access_tokens
SET revoked_at = @now::timestamptz, updated_at = @now
WHERE id = @id
  AND space_id = @space_id
  AND space_member_id = @space_member_id
  AND revoked_at IS NULL
RETURNING *;

-- name: ListUnrevokedPersonalAccessTokensBySpaceMember :many
-- スペースメンバー自身の、失効していない個人アクセストークンを新しい順に取得する。
-- 期限切れのトークンも含める。
SELECT * FROM personal_access_tokens
WHERE space_id = @space_id
  AND space_member_id = @space_member_id
  AND revoked_at IS NULL
ORDER BY created_at DESC, id DESC;
