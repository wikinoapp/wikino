package model

import (
	"time"
)

// PageAttachmentReference はページと添付ファイルの関連のドメインモデル
type PageAttachmentReference struct {
	ID           string
	AttachmentID string
	PageID       PageID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
