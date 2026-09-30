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
	Scopes        []Scope
	// Roleはトピックのロール。認可の切り替えまではScopesで判定する
	Role               TopicRole
	JoinedAt           time.Time
	LastPageModifiedAt *time.Time
}
