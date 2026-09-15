-- name: FindActiveSpaceMemberBySpaceAndUser :one
-- スペースIDとユーザーIDでアクティブなスペースメンバーを取得する
SELECT * FROM space_members WHERE space_id = $1 AND user_id = $2 AND active = true;

-- name: FindSpaceMembersByIDs :many
-- IDリストでスペースメンバーを一括取得する (スペースIDでスコープ)
SELECT * FROM space_members WHERE id = ANY($1::uuid[]) AND space_id = $2;

-- name: ListActiveSpaceMembersByUserAndSpaceIDs :many
-- ユーザーが複数スペースで持つアクティブなスペースメンバーを1クエリで一括取得する
-- (ホーム画面の参加中トピックでトピックごとの権限を解決する際のN+1を避けるために使用)。
SELECT * FROM space_members WHERE user_id = $1 AND space_id = ANY($2::uuid[]) AND active = true;
