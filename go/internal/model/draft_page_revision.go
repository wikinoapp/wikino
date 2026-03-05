package model

import (
	"time"
)

// DraftPageRevision は下書きページのバージョン（スナップショット）のドメインモデル
type DraftPageRevision struct {
	ID            DraftPageRevisionID
	DraftPageID   DraftPageID
	SpaceMemberID SpaceMemberID
	Title         string
	Body          string
	BodyHTML      string
	CreatedAt     time.Time
}
