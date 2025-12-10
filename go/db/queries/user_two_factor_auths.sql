-- name: GetUserTwoFactorAuthByUserID :one
-- ユーザーIDで二要素認証設定を取得する
SELECT * FROM user_two_factor_auths WHERE user_id = $1;

-- name: GetEnabledUserTwoFactorAuthByUserID :one
-- ユーザーIDで有効な二要素認証設定を取得する
SELECT * FROM user_two_factor_auths WHERE user_id = $1 AND enabled = true;

-- name: UpdateUserTwoFactorAuthRecoveryCodes :exec
-- リカバリーコードを更新する
UPDATE user_two_factor_auths
SET recovery_codes = $2, updated_at = $3
WHERE user_id = $1;
