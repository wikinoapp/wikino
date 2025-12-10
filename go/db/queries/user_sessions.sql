-- name: GetUserSessionByToken :one
-- セッショントークンでセッション情報を取得する
SELECT * FROM user_sessions WHERE token = $1;

-- name: GetUserSessionByID :one
-- セッションIDでセッション情報を取得する
SELECT * FROM user_sessions WHERE id = $1;

-- name: CreateUserSession :one
-- 新しいユーザーセッションを作成する
INSERT INTO user_sessions (
    user_id,
    token,
    ip_address,
    user_agent,
    signed_in_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: DeleteUserSession :exec
-- セッションを削除する
DELETE FROM user_sessions WHERE id = $1;

-- name: DeleteUserSessionByToken :exec
-- セッショントークンでセッションを削除する
DELETE FROM user_sessions WHERE token = $1;
