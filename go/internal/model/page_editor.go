package model

import (
	"time"
)

// PageEditor はページの編集者のドメインモデル
type PageEditor struct {
	ID                 string
	SpaceID            SpaceID
	PageID             PageID
	SpaceMemberID      SpaceMemberID
	LastPageModifiedAt time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
