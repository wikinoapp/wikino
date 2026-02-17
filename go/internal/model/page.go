package model

import (
	"time"
)

// Page はページのドメインモデル
type Page struct {
	ID                        PageID
	SpaceID                   SpaceID
	TopicID                   TopicID
	Number                    int32
	Title                     *string
	Body                      string
	BodyHTML                  string
	LinkedPageIDs             []PageID
	ModifiedAt                time.Time
	PublishedAt               *time.Time
	TrashedAt                 *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	PinnedAt                  *time.Time
	DiscardedAt               *time.Time
	FeaturedImageAttachmentID *AttachmentID
}
