package model

import (
	"time"
)

// TopicMember はトピックメンバーのドメインモデル
type TopicMember struct {
	ID                 TopicMemberID
	SpaceID            SpaceID
	TopicID            TopicID
	SpaceMemberID      SpaceMemberID
	Scopes             []Scope
	JoinedAt           time.Time
	LastPageModifiedAt *time.Time
}
