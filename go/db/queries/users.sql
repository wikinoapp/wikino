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

-- name: DiscardUser :execrows
-- ユーザーを論理削除する。既に論理削除したユーザーは対象にしない。
UPDATE users
SET discarded_at = @discarded_at, updated_at = @discarded_at
WHERE id = @id AND discarded_at IS NULL;

-- name: UndiscardUser :exec
-- 論理削除を取り消す。@discarded_atと同じ時刻に論理削除したユーザーだけを対象にし、
-- 別の削除の操作で論理削除し直したユーザーを戻さない。
UPDATE users
SET discarded_at = NULL, updated_at = @updated_at
WHERE id = @id AND discarded_at = @discarded_at;

-- name: LockDiscardedUser :one
-- 論理削除したユーザーを、物理削除が終わるまでロックする。
SELECT id FROM users
WHERE id = @id AND discarded_at IS NOT NULL
FOR UPDATE;

-- name: LockActiveUser :one
-- スペース作成・参加とアカウントの論理削除を直列化する。
-- 待機中に論理削除されたユーザーはロック対象にしない。
SELECT id FROM users
WHERE id = @id AND discarded_at IS NULL
FOR UPDATE;

-- name: DeleteDiscardedUser :execrows
-- 論理削除したユーザーを物理削除する。論理削除していないユーザーは対象にしない。
DELETE FROM users
WHERE id = @id AND discarded_at IS NOT NULL;
