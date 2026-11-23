-- name: GetEmailConfirmationByID :one
-- IDでメール確認情報を取得する
SELECT * FROM email_confirmations WHERE id = $1;

-- name: GetActiveEmailConfirmationByEmailAndEvent :one
-- メールアドレスとイベント種別で有効なメール確認情報を取得する
-- 有効条件: succeeded_at IS NULL (未確認) かつstarted_atが15分以内
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
    updated_at,
    user_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: ConsumeEmailUpdateConfirmation :one
-- 開始ユーザーとコードが一致する、期限内で未完了のメール変更の確認だけを完了する
UPDATE email_confirmations
SET succeeded_at = sqlc.arg(succeeded_at), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
  AND event = 1
  AND succeeded_at IS NULL
  AND started_at >= sqlc.arg(valid_after)
  AND UPPER(code) = UPPER(sqlc.arg(code)::text)
RETURNING *;

-- name: UpdateEmailConfirmationSucceededAt :exec
-- メール確認を完了状態に更新する
UPDATE email_confirmations
SET succeeded_at = $2, updated_at = $3
WHERE id = $1;

-- name: DeleteEmailConfirmationsStartedBefore :execrows
-- 指定された時刻より前に開始したメール確認情報を、確認の成否を問わず削除する
DELETE FROM email_confirmations
WHERE started_at < $1;
