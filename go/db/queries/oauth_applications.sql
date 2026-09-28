-- name: CreateOAuthApplication :one
-- スペースのOAuthアプリを作成する。公式クライアント (space_idがNULL) はこのクエリでは作らない
INSERT INTO oauth_applications (
    space_id,
    created_space_member_id,
    name,
    client_id,
    client_secret_digest,
    client_type,
    redirect_uris,
    created_at,
    updated_at
) VALUES (
    @space_id::uuid,
    @created_space_member_id::uuid,
    @name,
    @client_id,
    sqlc.narg('client_secret_digest'),
    @client_type,
    @redirect_uris::text[],
    @now,
    @now
) RETURNING *;

-- name: FindOAuthApplicationByClientID :one
-- クライアントIDで削除されていないOAuthアプリを取得する。
-- space_idを条件に含めない例外のクエリである。認可要求はスペースを特定する前の入口で、
-- スペースはこのアプリ (公式クライアントでは認可要求のresource) から決まるためである。
-- トークンの失効でも、トークンの有無によらず先にクライアントを認証するために使う。
-- client_idは全体で一意なので、別のスペースのアプリを取り違えることは無い。
-- 認可要求では、ここで得たspace_idを以降のクエリの条件に必ず含める。トークンの失効では、
-- トークンを取得した後でアプリがトークンのspace_idで使えるか確かめる。
SELECT * FROM oauth_applications
WHERE client_id = @client_id
  AND discarded_at IS NULL;

-- name: FindOAuthApplicationByClientIDAndSpaceID :one
-- クライアントIDで、スペースで使える削除されていないOAuthアプリを取得する。そのスペースの
-- アプリか、全スペース共通の公式クライアント (space_idがNULL) を返す
SELECT * FROM oauth_applications
WHERE client_id = @client_id
  AND (space_id = @space_id::uuid OR space_id IS NULL)
  AND discarded_at IS NULL;

-- name: FindOAuthApplicationByIDAndSpaceID :one
-- IDで、スペースの削除されていないOAuthアプリを取得する。公式クライアントはスペースに
-- 属さないため、このクエリでは見つからない
SELECT * FROM oauth_applications
WHERE id = @id
  AND space_id = @space_id::uuid
  AND discarded_at IS NULL;

-- name: ListOAuthApplicationsBySpace :many
-- スペースの削除されていないOAuthアプリを新しい順に取得する。公式クライアントは含めない
SELECT * FROM oauth_applications
WHERE space_id = @space_id::uuid
  AND discarded_at IS NULL
ORDER BY created_at DESC, id DESC;

-- name: ListOAuthApplicationsAvailableInSpaceByIDs :many
-- IDで、スペースで使える削除されていないOAuthアプリを取得する。そのスペースのアプリか、
-- 全スペース共通の公式クライアント (space_idがNULL) を返す。連携中のアプリの一覧で、許可の
-- アプリを引くために使う
SELECT * FROM oauth_applications
WHERE id = ANY(@ids::uuid[])
  AND (space_id = @space_id::uuid OR space_id IS NULL)
  AND discarded_at IS NULL;

-- name: UpdateOAuthApplication :one
-- スペースの削除されていないOAuthアプリの名前とリダイレクトURIを、フォームを開いた
-- 時点の版が一致する場合だけ更新する。対象が無いか版が異なる場合は行を返さない。
-- クライアントの種別は変えない
UPDATE oauth_applications
SET name = @name,
    redirect_uris = @redirect_uris::text[],
    updated_at = @now,
    version = version + 1
WHERE id = @id
  AND space_id = @space_id::uuid
  AND version = @expected_version
  AND discarded_at IS NULL
RETURNING *;

-- name: UpdateOAuthApplicationClientSecretDigest :one
-- スペースの削除されていないconfidentialクライアントのシークレットのダイジェストを置き換える。
-- 対象が無いか、publicクライアント (client_typeが1) の場合は行を返さない。以前のシークレットは
-- ダイジェストが残らないため、この時点から照合できなくなる
UPDATE oauth_applications
SET client_secret_digest = @client_secret_digest::text,
    updated_at = @now
WHERE id = @id
  AND space_id = @space_id::uuid
  AND client_type = 0
  AND discarded_at IS NULL
RETURNING *;

-- name: DiscardOAuthApplication :one
-- スペースの削除されていないOAuthアプリを削除し、そのアプリの許可と、許可に属するアクセス
-- トークン・リフレッシュトークンをすべて失効する。対象が無いか既に削除されている場合は
-- 何も変えずに行を返さない。
--
-- 失効を1文にまとめるのは、アプリだけが削除されてトークンが使える状態が途中にも残らない
-- ようにするためである。データを変更するCTEはどれも文の開始時点のスナップショットを見るため、
-- 許可・トークンの失効はアプリを削除できたとき (discardedに行があるとき) だけ行う
WITH discarded AS (
    UPDATE oauth_applications
    SET discarded_at = @now::timestamptz, updated_at = @now
    WHERE oauth_applications.id = @id
      AND oauth_applications.space_id = @space_id::uuid
      AND oauth_applications.discarded_at IS NULL
    RETURNING oauth_applications.*
),
target_grants AS (
    SELECT oauth_grants.id
    FROM oauth_grants
    INNER JOIN discarded ON oauth_grants.oauth_application_id = discarded.id
    WHERE oauth_grants.space_id = @space_id::uuid
),
revoked_grants AS (
    UPDATE oauth_grants
    SET revoked_at = @now::timestamptz, updated_at = @now
    WHERE oauth_grants.id IN (SELECT target_grants.id FROM target_grants)
      AND oauth_grants.space_id = @space_id::uuid
      AND oauth_grants.revoked_at IS NULL
),
revoked_access_tokens AS (
    UPDATE oauth_access_tokens
    SET revoked_at = @now::timestamptz, updated_at = @now
    WHERE oauth_access_tokens.oauth_grant_id IN (SELECT target_grants.id FROM target_grants)
      AND oauth_access_tokens.space_id = @space_id::uuid
      AND oauth_access_tokens.revoked_at IS NULL
),
revoked_refresh_tokens AS (
    UPDATE oauth_refresh_tokens
    SET revoked_at = @now::timestamptz, updated_at = @now
    WHERE oauth_refresh_tokens.oauth_grant_id IN (SELECT target_grants.id FROM target_grants)
      AND oauth_refresh_tokens.space_id = @space_id::uuid
      AND oauth_refresh_tokens.revoked_at IS NULL
)
SELECT * FROM discarded;
