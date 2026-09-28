-- name: CreateOAuthAuthorizationCode :one
-- 認可コードを作成する
INSERT INTO oauth_authorization_codes (
    oauth_grant_id,
    space_id,
    code_digest,
    scopes,
    redirect_uri,
    code_challenge,
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
-- 未使用の認可コードを使用済みにし、コードの許可・スコープでアクセストークンとリフレッシュ
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
        expires_at,
        created_at,
        updated_at
    )
    SELECT
        consumed.oauth_grant_id,
        consumed.space_id,
        @access_token_digest,
        consumed.scopes,
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
        expires_at,
        created_at,
        updated_at
    )
    SELECT
        consumed.oauth_grant_id,
        consumed.space_id,
        @refresh_token_digest,
        consumed.scopes,
        @refresh_token_expires_at,
        @now,
        @now
    FROM consumed
)
SELECT * FROM consumed;

-- name: CreateOAuthAuthorizationCodeWithGrant :one
-- 同意に応じて、スペースメンバーのOAuthアプリへの許可を作成または拡張し、その許可に属する
-- 認可コードを作成する。失効していない許可が既にあれば、許可のスコープを今回の要求との和に
-- 広げる (発行済みのトークンは以前の同意の範囲を持ち続けるため、狭めると許可の表示が実態と
-- ずれる)。1つの文で行うため、同じアプリへの同時の同意でも許可は1つに保たれ、許可だけが
-- 作られてコードが無い状態も残らない。
-- 許可のスコープの保存順は保証しない (新規作成では要求の順、拡張では文字列の順になる)。
-- 表示するときはmodel.APITokenScopesの順に並べ直す。
WITH upserted_grant AS (
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
    )
    ON CONFLICT (oauth_application_id, space_member_id) WHERE revoked_at IS NULL
    DO UPDATE SET
        scopes = ARRAY(
            SELECT DISTINCT merged.scope
            FROM unnest(oauth_grants.scopes || EXCLUDED.scopes) AS merged(scope)
            ORDER BY merged.scope
        ),
        updated_at = EXCLUDED.updated_at
    RETURNING id, space_id
)
INSERT INTO oauth_authorization_codes (
    oauth_grant_id,
    space_id,
    code_digest,
    scopes,
    redirect_uri,
    code_challenge,
    expires_at,
    created_at,
    updated_at
)
SELECT
    upserted_grant.id,
    upserted_grant.space_id,
    @code_digest,
    @scopes::text[],
    @redirect_uri,
    @code_challenge,
    @expires_at,
    @now,
    @now
FROM upserted_grant
RETURNING *;
