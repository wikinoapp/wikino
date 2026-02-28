package model

import (
	"time"
)

// SpaceMemberRole はスペースメンバーのロールを表す
type SpaceMemberRole int32

const (
	// SpaceMemberRoleOwner はオーナーロール
	SpaceMemberRoleOwner SpaceMemberRole = 0
	// SpaceMemberRoleMember はメンバーロール
	SpaceMemberRoleMember SpaceMemberRole = 1
)

// SpaceMember はスペースメンバーのドメインモデル
type SpaceMember struct {
	ID       SpaceMemberID
	SpaceID  SpaceID
	UserID   UserID
	Role     SpaceMemberRole
	JoinedAt time.Time
	Active   bool
}
