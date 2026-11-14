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

-- name: UpsertPendingUserTwoFactorAuth :one
-- 登録の途中の二要素認証設定 (enabled = false) を、新しい秘密鍵で作るか書き換える。
-- 既に有効な設定は書き換えず、行を返さない。
INSERT INTO user_two_factor_auths (user_id, secret, enabled, enabled_at, recovery_codes, created_at, updated_at)
VALUES (@user_id, @secret, false, NULL, '{}', @now, @now)
ON CONFLICT (user_id) DO UPDATE
SET secret = EXCLUDED.secret, enabled_at = NULL, recovery_codes = '{}', updated_at = EXCLUDED.updated_at
WHERE user_two_factor_auths.enabled = false
RETURNING *;

-- name: EnableUserTwoFactorAuth :execrows
-- 照合した秘密鍵の登録の途中の設定を有効にし、リカバリーコードを保存する。
-- 照合のあとに秘密鍵が差し替わった・既に有効になった場合は更新しない。
UPDATE user_two_factor_auths
SET enabled = true, enabled_at = @now::timestamp, recovery_codes = @recovery_codes, updated_at = @now
WHERE user_id = @user_id AND secret = @secret AND enabled = false;
