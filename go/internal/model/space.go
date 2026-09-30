package model

import (
	"time"
)

// Planはスペースの料金プランを表す
type Plan int32

const (
	// PlanFreeは無料プラン
	PlanFree Plan = 0
	// PlanSmallはスモールプラン
	PlanSmall Plan = 1
	// PlanLargeはラージプラン
	PlanLarge Plan = 2
)

// SpaceIdentifierMaxLengthはスペースの識別子の最大文字数。識別子はURLに使うため短く保つ
const SpaceIdentifierMaxLength = 20

// Spaceはスペースのドメインモデル
type Space struct {
	ID          SpaceID
	Identifier  SpaceIdentifier
	Name        string
	Plan        Plan
	JoinedAt    time.Time
	DiscardedAt *time.Time
}
