-- name: GetUserPasswordByUserID :one
-- ユーザーIDでパスワード情報を取得する
SELECT * FROM user_passwords WHERE user_id = $1;
