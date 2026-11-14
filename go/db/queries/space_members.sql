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

-- name: CreateSpaceMember :one
-- ユーザーをスペースに参加させる。scopesにはRails版が判定に使う、ロールに応じた値を書く。
INSERT INTO space_members (space_id, user_id, role, scopes, joined_at, created_at, updated_at)
VALUES (@space_id, @user_id, @role, @scopes, @now, @now, @now)
RETURNING *;

-- name: ExistsSpaceMemberByUser :one
-- ユーザーのスペースメンバーの行が残っているかを返す。参加をやめた (activeでない) 行と、
-- 削除済みのスペースの行も数える。どれもユーザーを外部キーで参照しているため。
-- スペースをまたいで数えるため、space_idはスペースとのJOINで確かめる。
SELECT EXISTS (
    SELECT 1 FROM space_members sm
    INNER JOIN spaces s ON s.id = sm.space_id
    WHERE sm.user_id = @user_id
);
