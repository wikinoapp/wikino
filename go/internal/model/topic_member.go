package model

import (
	"time"
)

// TopicMemberはトピックメンバーのドメインモデル
type TopicMember struct {
	ID            TopicMemberID
	SpaceID       SpaceID
	TopicID       TopicID
	SpaceMemberID SpaceMemberID
	// Roleは、このトピックで付けたロール。空なら、このトピックで追加する権限は無い
	Role               TopicRole
	JoinedAt           time.Time
	LastPageModifiedAt *time.Time
}
