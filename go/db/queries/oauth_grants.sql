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

-- name: LockOAuthGrantByID :one
-- 許可の失効とトークンの発行を直列化する。待機後は最新の状態 (失効の有無) を返す。
-- 外部キーの検査を妨げず、許可への更新どうしを排他にするためFOR NO KEY UPDATEを使う。
SELECT * FROM oauth_grants
WHERE id = @id
  AND space_id = @space_id
FOR NO KEY UPDATE;

-- name: ListUsableOAuthGrantsBySpaceMember :many
-- スペースメンバーが与えている失効していない許可のうち、有効なリフレッシュトークン (未使用・未失効・
-- 期限内) を持つものを新しい順に取得する。連携の一覧に出す許可で、交換されなかった許可や、
-- リフレッシュトークンの期限が切れた許可は、もう使えないため出さない
SELECT * FROM oauth_grants
WHERE oauth_grants.space_id = @space_id
  AND oauth_grants.space_member_id = @space_member_id
  AND oauth_grants.revoked_at IS NULL
  AND EXISTS (
      SELECT 1 FROM oauth_refresh_tokens
      WHERE oauth_refresh_tokens.oauth_grant_id = oauth_grants.id
        AND oauth_refresh_tokens.space_id = @space_id
        AND oauth_refresh_tokens.used_at IS NULL
        AND oauth_refresh_tokens.revoked_at IS NULL
        AND oauth_refresh_tokens.expires_at > @now::timestamptz
  )
ORDER BY oauth_grants.created_at DESC, oauth_grants.id DESC;

-- name: UpdateOAuthGrantLastUsedAt :exec
-- 許可の最終利用日時をnowにする。最終利用日時がstale_beforeより古い (またはまだ無い) ときだけ
-- 更新し、アクセストークンの照合のたびに書き込まないようにする。利用者による変更ではないため、
-- updated_atは更新しない
UPDATE oauth_grants
SET last_used_at = @now::timestamptz
WHERE id = @id
  AND space_id = @space_id
  AND revoked_at IS NULL
  AND (last_used_at IS NULL OR last_used_at < @stale_before::timestamptz);

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

-- name: EvictOAuthGrants :exec
-- 同じアプリ・メンバーに許可を1つ作る前に、失効していない許可が@max_grants - 1件以下になるまで、
-- 使われていないものから順に許可と、許可に属するアクセストークン・リフレッシュトークンを失効する。
-- 押し出す順は次のとおりで、同じ順位の中では古いものから選ぶ。
--   1. まだ使われておらず (last_used_atがNULL)、@grace_beforeより前に作ったもの (交換されなかった同意)
--   2. 最終利用日時が最も古いもの
--   3. まだ使われていない、@grace_before以降に作ったもの (ログインの途中の同意)
-- 同時の作成は直列化しないため、一時的に上限を超えうるが、次の作成で押し出す。
-- 失効は1文で行い、許可だけが失効してトークンが使える状態は途中にも残らない
WITH unrevoked AS (
    SELECT oauth_grants.id, oauth_grants.last_used_at, oauth_grants.created_at
    FROM oauth_grants
    WHERE oauth_grants.oauth_application_id = @oauth_application_id
      AND oauth_grants.space_id = @space_id
      AND oauth_grants.space_member_id = @space_member_id
      AND oauth_grants.revoked_at IS NULL
),
victims AS (
    SELECT unrevoked.id
    FROM unrevoked
    ORDER BY
        CASE
            WHEN unrevoked.last_used_at IS NULL AND unrevoked.created_at < @grace_before::timestamptz THEN 0
            WHEN unrevoked.last_used_at IS NOT NULL THEN 1
            ELSE 2
        END,
        COALESCE(unrevoked.last_used_at, unrevoked.created_at),
        unrevoked.id
    LIMIT (SELECT GREATEST(count(*) + 1 - sqlc.arg(max_grants)::bigint, 0) FROM unrevoked)
),
evicted_grants AS (
    UPDATE oauth_grants
    SET revoked_at = @now::timestamptz, updated_at = @now
    WHERE oauth_grants.id IN (SELECT victims.id FROM victims)
      AND oauth_grants.space_id = @space_id
      AND oauth_grants.revoked_at IS NULL
    RETURNING oauth_grants.id
),
evicted_access_tokens AS (
    UPDATE oauth_access_tokens
    SET revoked_at = @now::timestamptz, updated_at = @now
    WHERE oauth_access_tokens.oauth_grant_id IN (SELECT evicted_grants.id FROM evicted_grants)
      AND oauth_access_tokens.space_id = @space_id
      AND oauth_access_tokens.revoked_at IS NULL
)
UPDATE oauth_refresh_tokens
SET revoked_at = @now::timestamptz, updated_at = @now
WHERE oauth_refresh_tokens.oauth_grant_id IN (SELECT evicted_grants.id FROM evicted_grants)
  AND oauth_refresh_tokens.space_id = @space_id
  AND oauth_refresh_tokens.revoked_at IS NULL;
