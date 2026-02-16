package model

import (
	"time"
)

// TopicMemberRole はトピックメンバーのロールを表す
type TopicMemberRole int32

const (
	// TopicMemberRoleAdmin は管理者ロール
	TopicMemberRoleAdmin TopicMemberRole = 0
	// TopicMemberRoleMember はメンバーロール
	TopicMemberRoleMember TopicMemberRole = 1
)

// TopicMember はトピックメンバーのドメインモデル
type TopicMember struct {
	ID                 TopicMemberID
	SpaceID            SpaceID
	TopicID            TopicID
	SpaceMemberID      SpaceMemberID
	Role               TopicMemberRole
	JoinedAt           time.Time
	LastPageModifiedAt *time.Time
}
