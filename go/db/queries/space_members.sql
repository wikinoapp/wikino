-- name: FindActiveSpaceMemberBySpaceAndUser :one
-- スペースIDとユーザーIDでアクティブなスペースメンバーを取得する
SELECT * FROM space_members WHERE space_id = $1 AND user_id = $2 AND active = true;
