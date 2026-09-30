package model

import (
	"time"
)

// SpaceMemberはスペースメンバーのドメインモデル
type SpaceMember struct {
	ID      SpaceMemberID
	SpaceID SpaceID
	UserID  UserID
	Scopes  []Scope
	// Roleはスペースのロール。認可の切り替えまではScopesで判定する
	Role     SpaceRole
	JoinedAt time.Time
	Active   bool
}
