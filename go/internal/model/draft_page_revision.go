package model

import (
	"time"
)

// DraftPageRevisionは下書きページのバージョン (スナップショット) のドメインモデル
type DraftPageRevision struct {
	ID            DraftPageRevisionID
	DraftPageID   DraftPageID
	SpaceID       SpaceID
	SpaceMemberID SpaceMemberID
	Title         string
	Body          string
	CreatedAt     time.Time
}
