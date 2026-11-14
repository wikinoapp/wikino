-- name: GetUserPasswordByUserID :one
-- ユーザーIDでパスワード情報を取得する
SELECT * FROM user_passwords WHERE user_id = $1;

-- name: CreateUserPassword :one
-- 新しいユーザーパスワードを作成する
INSERT INTO user_passwords (
    user_id,
    password_digest,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: UpdateUserPasswordDigest :exec
-- ユーザーIDでパスワードダイジェストを更新する
UPDATE user_passwords
SET
    password_digest = $2,
    updated_at = $3
WHERE user_id = $1;

-- name: DeleteUserPasswordByUserID :exec
-- ユーザーのパスワードを削除する
DELETE FROM user_passwords WHERE user_id = @user_id;
