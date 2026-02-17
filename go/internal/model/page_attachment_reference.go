package model

import (
	"time"
)

// PageAttachmentReference はページと添付ファイルの関連のドメインモデル
type PageAttachmentReference struct {
	ID           PageAttachmentReferenceID
	AttachmentID AttachmentID
	PageID       PageID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
