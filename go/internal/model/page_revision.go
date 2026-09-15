package model

import (
	"time"
)

// PageRevisionは公開されたページのスナップショットのドメインモデル
type PageRevision struct {
	ID            PageRevisionID
	SpaceID       SpaceID
	SpaceMemberID SpaceMemberID
	PageID        PageID
	Title         string
	Body          string
	BodyHTML      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
