-- name: GetPasswordResetTokenByTokenDigest :one
-- トークンダイジェストでパスワードリセットトークンを取得する
SELECT * FROM password_reset_tokens WHERE token_digest = $1;

-- name: GetUnusedPasswordResetTokensByUserID :many
-- ユーザーIDで未使用のパスワードリセットトークンを取得する
SELECT * FROM password_reset_tokens
WHERE user_id = $1
  AND used_at IS NULL;

-- name: CreatePasswordResetToken :one
-- 新しいパスワードリセットトークンを作成する
INSERT INTO password_reset_tokens (
    user_id,
    token_digest,
    expires_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdatePasswordResetTokenUsedAt :exec
-- パスワードリセットトークンを使用済みに更新する
UPDATE password_reset_tokens
SET used_at = $2, updated_at = $3
WHERE id = $1;

-- name: DeleteUnusedPasswordResetTokensByUserID :exec
-- ユーザーIDで未使用のパスワードリセットトークンを削除する
DELETE FROM password_reset_tokens
WHERE user_id = $1
  AND used_at IS NULL;
