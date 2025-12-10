-- name: GetUserByID :one
-- ユーザーをIDで取得する
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
-- ユーザーをメールアドレスで取得する（削除されていないユーザーのみ）
SELECT * FROM users WHERE email = $1 AND discarded_at IS NULL;

-- name: GetUserByAtname :one
-- ユーザーをアットネーム（@ユーザー名）で取得する（削除されていないユーザーのみ）
SELECT * FROM users WHERE atname = $1 AND discarded_at IS NULL;
