-- name: CreateOAuthDeviceAuthorization :one
-- デバイス認可の要求を許可待ちで作成する。端末名 (@device_name) は送られなかったときNULLを渡す
INSERT INTO oauth_device_authorizations (
    oauth_application_id,
    space_id,
    device_code_digest,
    user_code_digest,
    scopes,
    interval_seconds,
    device_name,
    expires_at,
    created_at,
    updated_at
) VALUES (
    @oauth_application_id,
    @space_id,
    @device_code_digest,
    @user_code_digest,
    @scopes::text[],
    @interval_seconds,
    sqlc.narg(device_name),
    @expires_at,
    @now,
    @now
) RETURNING *;

-- name: FindOAuthDeviceAuthorizationByDeviceCodeDigest :one
-- デバイスコードのダイジェストでデバイス認可の要求を取得する。
-- space_idを条件に含めない例外のクエリである。トークン要求はスペースを特定する前の入口で、
-- スペースはこの要求から決まるためである。呼び出し側は、ここで得たspace_idを以降の
-- クエリの条件に必ず含める。
SELECT * FROM oauth_device_authorizations WHERE device_code_digest = @device_code_digest;

-- name: FindOAuthDeviceAuthorizationByUserCodeDigest :one
-- ユーザーコードのダイジェストでデバイス認可の要求を取得する。
-- space_idを条件に含めない例外のクエリである。ユーザーコードの入力はスペースを特定する前の
-- 入口で、スペースはこの要求から決まるためである。呼び出し側は、ここで得たspace_idを以降の
-- クエリの条件に必ず含める。
SELECT * FROM oauth_device_authorizations WHERE user_code_digest = @user_code_digest;

-- name: LockOAuthDeviceAuthorizationByID :one
-- 交換前に要求をロックし、ロック待機後の状態と期限を呼び出し側で確認する。
-- 交換済みも返し、期限切れと再提示を区別できるようにする
SELECT * FROM oauth_device_authorizations
WHERE id = @id
  AND space_id = @space_id
FOR UPDATE;

-- name: LockPendingOAuthDeviceAuthorization :one
-- 許可待ちで期限内のデバイス認可の要求をロックして取得する。同じ要求への許可・拒否の同時の
-- 送信を直列化し、ロック待機後の状態で許可できるかを判断するために使う。
-- @nowはロック待機前の時刻で、待機後の再評価にも同じ値を使うため、待機中に期限が切れた要求も
-- 返りうる。待機後の期限は呼び出し側が確かめる
SELECT * FROM oauth_device_authorizations
WHERE id = @id
  AND space_id = @space_id
  AND status = 0
  AND expires_at > @now::timestamptz
FOR UPDATE;

-- name: DenyOAuthDeviceAuthorization :one
-- 許可待ちで期限内のデバイス認可の要求を拒否済みにする。許可待ちでなくなった要求と期限切れの
-- 要求は更新せず、行を返さない。
-- @nowはロック待機前の時刻のため、同じ要求への許可の送信を待つ間に期限が切れた要求も拒否済みに
-- なりうる。拒否はアプリに権限を与えず、利用者も拒否を選んでいるため、待機後の期限は確かめない
UPDATE oauth_device_authorizations
SET status = 2, updated_at = @now
WHERE id = @id
  AND space_id = @space_id
  AND status = 0
  AND expires_at > @now::timestamptz
RETURNING *;

-- name: ApproveOAuthDeviceAuthorizationWithGrant :one
-- 利用者の許可に応じて、スペースメンバーのOAuthアプリへの許可を新しく作り、デバイス認可の要求を
-- 許可済みにして許可を結び付ける。許可は認可コードの発行 (CreateOAuthAuthorizationCodeWithGrant)
-- と同じく要求ごとに作り、同じアプリ・メンバーの既存の許可には触れない。
-- 要求のスコープは利用者が選んだ範囲に置き換え、デバイスコードの交換で発行するトークンの
-- スコープにする。同じTxの先行する文で、要求をLockPendingOAuthDeviceAuthorizationでロックしておく。
-- 許可の端末名は、要求で受け取った端末名を写す。
-- 要求が許可待ちでなければ許可を作らず、更新する行も無く、文全体が行を返さない。
WITH created_grant AS (
    INSERT INTO oauth_grants (
        oauth_application_id,
        space_id,
        space_member_id,
        scopes,
        device_name,
        created_at,
        updated_at
    )
    SELECT
        @oauth_application_id,
        @space_id,
        @space_member_id,
        @scopes::text[],
        oauth_device_authorizations.device_name,
        @now,
        @now
    FROM oauth_device_authorizations
    WHERE oauth_device_authorizations.id = @id
      AND oauth_device_authorizations.space_id = @space_id
      AND oauth_device_authorizations.status = 0
    RETURNING id
)
UPDATE oauth_device_authorizations
SET
    status = 1,
    oauth_grant_id = created_grant.id,
    scopes = @scopes::text[],
    updated_at = @now
FROM created_grant
WHERE oauth_device_authorizations.id = @id
  AND oauth_device_authorizations.space_id = @space_id
  AND oauth_device_authorizations.status = 0
RETURNING oauth_device_authorizations.*;

-- name: RecordOAuthDeviceAuthorizationPoll :one
-- 許可待ちのデバイス認可の要求へのポーリングを記録し、前回のポーリングから間隔を空けずに
-- 届いたか (too_fast) を返す。間隔を空けずに届いたときは、以降の間隔を@slow_down_secondsだけ
-- 延ばす (RFC 8628 §3.5の `slow_down`)。前回の日時と間隔はロックして読み、同時に届いた
-- ポーリングでも延長を取りこぼさない。許可待ちでない要求は更新せず、行を返さない
WITH previous AS (
    SELECT
        oauth_device_authorizations.id AS previous_id,
        oauth_device_authorizations.last_polled_at AS previous_polled_at,
        oauth_device_authorizations.interval_seconds AS previous_interval_seconds
    FROM oauth_device_authorizations
    WHERE oauth_device_authorizations.id = @id
      AND oauth_device_authorizations.space_id = @space_id
      AND oauth_device_authorizations.status = 0
    FOR UPDATE
),
judged AS (
    SELECT
        previous_id,
        previous_interval_seconds,
        COALESCE(previous_polled_at + make_interval(secs => previous_interval_seconds) > @now::timestamptz, FALSE) AS too_fast
    FROM previous
)
UPDATE oauth_device_authorizations
SET
    interval_seconds = CASE
        WHEN judged.too_fast THEN judged.previous_interval_seconds + @slow_down_seconds::integer
        ELSE judged.previous_interval_seconds
    END,
    last_polled_at = @now::timestamptz,
    updated_at = @now
FROM judged
WHERE oauth_device_authorizations.id = judged.previous_id
  AND oauth_device_authorizations.space_id = @space_id
RETURNING judged.too_fast::boolean AS too_fast;

-- name: ExchangeOAuthDeviceAuthorization :one
-- 許可済みのデバイス認可の要求を交換済みにし、要求の許可・スコープでアクセストークンと
-- リフレッシュトークンを作成して、交換済みにした要求を返す。認可コードの交換
-- (ExchangeOAuthAuthorizationCode) と同じく、1つの文で判定と更新を行うため、同じデバイスコードの
-- 同時の交換でもトークンを作るのは1回だけである。許可済みでなければ何もせず行を返さない。
-- 有効期限は呼び出し側が交換の前に確かめる。トークンの宛先 (@audience) は呼び出し側が決める
WITH consumed AS (
    UPDATE oauth_device_authorizations
    SET status = 3, updated_at = @now
    WHERE oauth_device_authorizations.id = @id
      AND oauth_device_authorizations.space_id = @space_id
      AND oauth_device_authorizations.status = 1
    RETURNING oauth_device_authorizations.*
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
        @audience::integer,
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
        @audience::integer,
        @refresh_token_expires_at,
        @now,
        @now
    FROM consumed
)
SELECT * FROM consumed;
