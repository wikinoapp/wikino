package model

import (
	"time"
)

// DraftPage はページの下書きのドメインモデル
type DraftPage struct {
	ID            DraftPageID
	SpaceID       SpaceID
	PageID        PageID
	SpaceMemberID SpaceMemberID
	TopicID       TopicID
	Title         *string
	Body          string
	BodyHTML      string
	LinkedPageIDs []PageID
	ModifiedAt    time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Page  *Page
	Topic *Topic
}
