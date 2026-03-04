-- name: FindTopicMemberBySpaceMemberAndTopic :one
-- スペースメンバーIDとトピックIDでトピックメンバーを取得する
SELECT * FROM topic_members WHERE space_member_id = $1 AND topic_id = $2 AND space_id = $3;

-- name: UpdateTopicMemberLastPageModifiedAt :exec
-- トピックメンバーのlast_page_modified_atを更新する（ページ公開時に使用）
UPDATE topic_members SET last_page_modified_at = $1, updated_at = $2 WHERE topic_id = $3 AND space_member_id = $4 AND space_id = $5;
