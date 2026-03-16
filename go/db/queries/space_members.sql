-- name: FindActiveSpaceMemberBySpaceAndUser :one
-- スペースIDとユーザーIDでアクティブなスペースメンバーを取得する
SELECT * FROM space_members WHERE space_id = $1 AND user_id = $2 AND active = true;

-- name: FindSpaceMembersByIDs :many
-- IDリストでスペースメンバーを一括取得する（スペースIDでスコープ）
SELECT * FROM space_members WHERE id = ANY($1::uuid[]) AND space_id = $2;
