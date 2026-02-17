package model

import (
	"time"
)

// Plan はスペースの料金プランを表す
type Plan int32

const (
	// PlanFree は無料プラン
	PlanFree Plan = 0
	// PlanSmall はスモールプラン
	PlanSmall Plan = 1
	// PlanLarge はラージプラン
	PlanLarge Plan = 2
)

// Space はスペースのドメインモデル
type Space struct {
	ID          SpaceID
	Identifier  string
	Name        string
	Plan        Plan
	JoinedAt    time.Time
	DiscardedAt *time.Time
}
