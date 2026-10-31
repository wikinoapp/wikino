-- name: CreateOAuthAuthorizationCode :one
-- 認可コードを作成する。@audienceは交換で発行するトークンに引き継ぐ宛先である
INSERT INTO oauth_authorization_codes (
    oauth_grant_id,
    space_id,
    code_digest,
    scopes,
    redirect_uri,
    code_challenge,
    audience,
    expires_at,
    created_at,
    updated_at
) VALUES (
    @oauth_grant_id,
    @space_id,
    @code_digest,
    @scopes::text[],
    @redirect_uri,
    @code_challenge,
    @audience,
    @expires_at,
    @now,
    @now
) RETURNING *;

-- name: FindOAuthAuthorizationCodeByCodeDigest :one
-- コードダイジェストで認可コードを取得する。
-- space_idを条件に含めない例外のクエリである。トークン要求はスペースを特定する前の入口で、
-- スペースはこの認可コードから決まるためである。呼び出し側は、ここで得たspace_idを以降の
-- クエリの条件に必ず含める。
SELECT * FROM oauth_authorization_codes WHERE code_digest = @code_digest;

-- name: ExchangeOAuthAuthorizationCode :one
-- 未使用の認可コードを使用済みにし、コードの許可・スコープ・宛先でアクセストークンとリフレッシュ
-- トークンを作成して、使用済みにしたコードを返す。1つの文で判定と更新を行うため、同じコードの
-- 同時の交換でもトークンを作るのは1回だけである。既に使用済みなら何もせず行を返さない。
-- 有効期限・リダイレクトURI・PKCEは呼び出し側が交換の前に確かめる。
--
-- コードの消費とトークンの作成を1文にまとめるのは、コードだけが使用済みになりトークンが
-- 作られない状態を残さないためである。その状態でクライアントが同じコードを再び送ると、
-- 再利用として許可のトークンがすべて失効してしまう。データを変更するCTEは参照されなくても
-- 実行され、どれも使用済みにできたコード (consumedの行) があるときだけ行を作る
WITH consumed AS (
    UPDATE oauth_authorization_codes
    SET used_at = @now::timestamptz, updated_at = @now
    WHERE oauth_authorization_codes.id = @id
      AND oauth_authorization_codes.space_id = @space_id
      AND oauth_authorization_codes.used_at IS NULL
    RETURNING oauth_authorization_codes.*
),
created_access_token AS (
    INSERT INTO oauth_access_tokens (
        oauth_grant_id,
        space_id,
        token_digest,
        scopes,
        audience,
        expires_at,
        created_at,
        updated_at
    )
    SELECT
        consumed.oauth_grant_id,
        consumed.space_id,
        @access_token_digest,
        consumed.scopes,
        consumed.audience,
        @access_token_expires_at,
        @now,
        @now
    FROM consumed
),
created_refresh_token AS (
    INSERT INTO oauth_refresh_tokens (
        oauth_grant_id,
        space_id,
        token_digest,
        scopes,
        audience,
        expires_at,
        created_at,
        updated_at
    )
    SELECT
        consumed.oauth_grant_id,
        consumed.space_id,
        @refresh_token_digest,
        consumed.scopes,
        consumed.audience,
        @refresh_token_expires_at,
        @now,
        @now
    FROM consumed
)
SELECT * FROM consumed;

-- name: CreateOAuthAuthorizationCodeWithGrant :one
-- 同意に応じて、スペースメンバーのOAuthアプリへの許可を新しく作り、その許可に属する認可コードを
-- 作成する。許可は同意ごとに作り、同じアプリ・メンバーの既存の許可 (別の端末のログインなど) には
-- 触れない。1つの文で行うため、許可だけが作られてコードが無い状態は残らない。
-- 許可のスコープの保存順は保証しない。表示するときはmodel.APITokenScopesの順に並べ直す。
-- 端末名 (@device_name) は認可要求で送られたもので、送られなかったときはNULLを渡す。
-- @audienceは、交換で発行するトークンに引き継ぐ宛先である。
WITH created_grant AS (
    INSERT INTO oauth_grants (
        oauth_application_id,
        space_id,
        space_member_id,
        scopes,
        device_name,
        created_at,
        updated_at
    ) VALUES (
        @oauth_application_id,
        @space_id,
        @space_member_id,
        @scopes::text[],
        sqlc.narg(device_name),
        @now,
        @now
    )
    RETURNING id, space_id
)
INSERT INTO oauth_authorization_codes (
    oauth_grant_id,
    space_id,
    code_digest,
    scopes,
    redirect_uri,
    code_challenge,
    audience,
    expires_at,
    created_at,
    updated_at
)
SELECT
    created_grant.id,
    created_grant.space_id,
    @code_digest,
    @scopes::text[],
    @redirect_uri,
    @code_challenge,
    @audience,
    @expires_at,
    @now,
    @now
FROM created_grant
RETURNING *;
