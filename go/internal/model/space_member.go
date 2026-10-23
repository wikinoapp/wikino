package model

import (
	"time"
)

// SpaceMemberはスペースメンバーのドメインモデル
type SpaceMember struct {
	ID      SpaceMemberID
	SpaceID SpaceID
	UserID  UserID
	// Roleは、スペースのロール。権限はロールから引き、space_members.scopesは読まない
	Role SpaceRole
	// MaxTopicRoleは、トピックで付けたロールの上限。nilなら制限しない
	MaxTopicRole *TopicRole
	JoinedAt     time.Time
	Active       bool
}
