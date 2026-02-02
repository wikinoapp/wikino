-- name: GetUserByID :one
-- ユーザーをIDで取得する
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
-- ユーザーをメールアドレスで取得する（削除されていないユーザーのみ）
SELECT * FROM users WHERE email = $1 AND discarded_at IS NULL;

-- name: GetUserByAtname :one
-- ユーザーをアットネーム（@ユーザー名）で取得する（削除されていないユーザーのみ）
SELECT * FROM users WHERE atname = $1 AND discarded_at IS NULL;

-- name: CreateUser :one
-- 新しいユーザーを作成する
INSERT INTO users (
    email,
    atname,
    name,
    description,
    locale,
    time_zone,
    joined_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;
