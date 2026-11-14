-- name: GetUserByID :one
-- ユーザーをIDで取得する
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
-- ユーザーをメールアドレスで取得する (削除されていないユーザーのみ)
SELECT * FROM users WHERE email = $1 AND discarded_at IS NULL;

-- name: GetUserByAtname :one
-- ユーザーをアットネーム (@ユーザー名) で取得する (削除されていないユーザーのみ)
SELECT * FROM users WHERE atname = $1 AND discarded_at IS NULL;

-- name: FindUsersByIDs :many
-- IDリストでユーザーを一括取得する
SELECT * FROM users WHERE id = ANY($1::uuid[]);

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

-- name: ExistsUserByAtname :one
-- アットネームを使っているユーザーがいるかを返す。一意インデックスは削除済みのユーザーも
-- 含むため、削除済みのユーザーも数える
SELECT EXISTS (SELECT 1 FROM users WHERE atname = $1);

-- name: ExistsOtherUserByAtname :one
-- 指定したユーザー以外に、アットネームを使っているユーザーがいるかを返す (削除済みのユーザーも数える)
SELECT EXISTS (SELECT 1 FROM users WHERE atname = @atname AND id <> @user_id);

-- name: UpdateUserProfile :one
-- ユーザーのアットネーム・名前・自己紹介を更新する (削除されていないユーザーのみ)
UPDATE users
SET atname = @atname, name = @name, description = @description, updated_at = @now
WHERE id = @id AND discarded_at IS NULL
RETURNING *;

-- name: ExistsUserByEmail :one
-- メールアドレスを使っているユーザーがいるかを返す。一意インデックスは削除済みのユーザーも
-- 含むため、削除済みのユーザーも数える
SELECT EXISTS (SELECT 1 FROM users WHERE email = $1);

-- name: UpdateUserEmail :one
-- ユーザーのメールアドレスを更新する (削除されていないユーザーのみ)
UPDATE users
SET email = @email, updated_at = @now
WHERE id = @id AND discarded_at IS NULL
RETURNING *;
