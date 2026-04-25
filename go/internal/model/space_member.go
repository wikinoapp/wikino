package model

import (
	"time"
)

// SpaceMember はスペースメンバーのドメインモデル
type SpaceMember struct {
	ID       SpaceMemberID
	SpaceID  SpaceID
	UserID   UserID
	Scopes   []Scope
	JoinedAt time.Time
	Active   bool
}
