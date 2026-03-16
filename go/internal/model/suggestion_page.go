package model

import (
	"time"
)

// SuggestionPage は編集提案ページのドメインモデル
type SuggestionPage struct {
	ID             SuggestionPageID
	SpaceID        SpaceID
	SuggestionID   SuggestionID
	PageID         PageID
	PageRevisionID PageRevisionID
	Title          *string
	Body           string
	BodyHTML       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
