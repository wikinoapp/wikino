-- name: GetEmailConfirmationByID :one
-- IDでメール確認情報を取得する
SELECT * FROM email_confirmations WHERE id = $1;

-- name: GetActiveEmailConfirmationByEmailAndEvent :one
-- メールアドレスとイベント種別で有効なメール確認情報を取得する
-- 有効条件: succeeded_at IS NULL（未確認）かつ started_at が15分以内
SELECT * FROM email_confirmations
WHERE email = $1
  AND event = $2
  AND succeeded_at IS NULL
  AND started_at > NOW() - INTERVAL '15 minutes'
ORDER BY started_at DESC
LIMIT 1;

-- name: CreateEmailConfirmation :one
-- 新しいメール確認情報を作成する
INSERT INTO email_confirmations (
    email,
    event,
    code,
    started_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateEmailConfirmationSucceededAt :exec
-- メール確認を完了状態に更新する
UPDATE email_confirmations
SET succeeded_at = $2, updated_at = $3
WHERE id = $1;
