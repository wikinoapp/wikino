package model

import (
	"time"
)

// SpaceMemberはスペースメンバーのドメインモデル
type SpaceMember struct {
	ID       SpaceMemberID
	SpaceID  SpaceID
	UserID   UserID
	Scopes   []Scope
	JoinedAt time.Time
	Active   bool
}
