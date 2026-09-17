package model

import (
	"time"
)

// PageEditorはページの編集者のドメインモデル
type PageEditor struct {
	ID                 PageEditorID
	SpaceID            SpaceID
	PageID             PageID
	SpaceMemberID      SpaceMemberID
	LastPageModifiedAt time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
