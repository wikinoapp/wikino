-- name: CreateOAuthRefreshToken :one
-- OAuthのリフレッシュトークンを作成する。認可コードの交換で最初のトークンを発行するときに使い、
-- ローテーションではRotateOAuthRefreshTokenを使う
INSERT INTO oauth_refresh_tokens (
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

-- name: FindOAuthRefreshTokenByTokenDigest :one
-- トークンダイジェストでOAuthのリフレッシュトークンを取得する。
-- space_idを条件に含めない例外のクエリである。トークン要求はスペースを特定する前の入口で、
-- スペースはこのトークンから決まるためである。呼び出し側は、ここで得たspace_idを以降の
-- クエリの条件に必ず含める。
SELECT * FROM oauth_refresh_tokens WHERE token_digest = @token_digest;

-- name: RotateOAuthRefreshToken :one
-- リフレッシュトークンを使用済みにし、同じ許可・スコープの新しいリフレッシュトークンと、
-- 同じ許可のアクセストークンを作成して、新しいリフレッシュトークンを返す。1つの文で判定と
-- 更新を行うため、同じトークンの同時の更新でも新しいトークンを返すのは1回だけである。
-- 元のトークンが使用済み・失効済み・期限切れなら何もせず行を返さない。
--
-- アクセストークンのスコープは、元のスコープを狭めた要求 (RFC 6749 §6) に応じて呼び出し側が
-- 決める。アクセストークンの作成も1文に含めるのは、リフレッシュトークンだけがローテーション
-- され、クライアントが新しいトークンを受け取れない状態を残さないためである。その状態で
-- クライアントが元のトークンを再び送ると、再利用として許可のトークンがすべて失効してしまう
WITH used AS (
    UPDATE oauth_refresh_tokens
    SET used_at = @now::timestamptz, updated_at = @now
    WHERE oauth_refresh_tokens.id = @id
      AND oauth_refresh_tokens.space_id = @space_id
      AND oauth_refresh_tokens.used_at IS NULL
      AND oauth_refresh_tokens.revoked_at IS NULL
      AND oauth_refresh_tokens.expires_at > @now::timestamptz
    RETURNING oauth_refresh_tokens.id, oauth_refresh_tokens.oauth_grant_id, oauth_refresh_tokens.space_id, oauth_refresh_tokens.scopes
),
created_access_token AS (
    INSERT INTO oauth_access_tokens (
        oauth_grant_id,
        space_id,
        token_digest,
        scopes,
        expires_at,
        created_at,
        updated_at
    )
    SELECT
        used.oauth_grant_id,
        used.space_id,
        @access_token_digest,
        @access_token_scopes::text[],
        @access_token_expires_at,
        @now,
        @now
    FROM used
)
INSERT INTO oauth_refresh_tokens (
    oauth_grant_id,
    space_id,
    token_digest,
    scopes,
    previous_refresh_token_id,
    expires_at,
    created_at,
    updated_at
)
SELECT
    used.oauth_grant_id,
    used.space_id,
    @token_digest,
    used.scopes,
    used.id,
    @expires_at,
    @now,
    @now
FROM used
RETURNING *;

-- name: RevokeOAuthRefreshTokensByOAuthGrant :exec
-- 許可に属する失効していないリフレッシュトークンをすべて失効する
UPDATE oauth_refresh_tokens
SET revoked_at = @now::timestamptz, updated_at = @now
WHERE oauth_grant_id = @oauth_grant_id
  AND space_id = @space_id
  AND revoked_at IS NULL;

-- name: RevokeOAuthRefreshToken :exec
-- リフレッシュトークンを1つ失効し、同じ許可のアクセストークンをすべて失効する (RFC 7009 §2.1)。
-- 同じ許可のほかのリフレッシュトークン (同じアプリを別の端末で使っているもの) は失効しない。
-- アクセストークンはどのリフレッシュトークンから発行したかを持たないため、許可の単位で失効する。
-- ほかの端末はリフレッシュトークンで新しいアクセストークンを得られる。
-- 元のトークンが使用済み・失効済み・期限切れなら何も変えない
WITH revoked AS (
    UPDATE oauth_refresh_tokens
    SET revoked_at = @now::timestamptz, updated_at = @now
    WHERE oauth_refresh_tokens.id = @id
      AND oauth_refresh_tokens.space_id = @space_id
      AND oauth_refresh_tokens.used_at IS NULL
      AND oauth_refresh_tokens.revoked_at IS NULL
      AND oauth_refresh_tokens.expires_at > @now::timestamptz
    RETURNING oauth_refresh_tokens.oauth_grant_id
)
UPDATE oauth_access_tokens
SET revoked_at = @now::timestamptz, updated_at = @now
WHERE oauth_access_tokens.oauth_grant_id IN (SELECT revoked.oauth_grant_id FROM revoked)
  AND oauth_access_tokens.space_id = @space_id
  AND oauth_access_tokens.revoked_at IS NULL;
