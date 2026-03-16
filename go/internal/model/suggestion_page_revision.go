package model

import (
	"time"
)

// SuggestionPageRevision は編集提案ページリビジョンのドメインモデル
type SuggestionPageRevision struct {
	ID                  SuggestionPageRevisionID
	SpaceID             SpaceID
	SuggestionPageID    SuggestionPageID
	EditorSpaceMemberID SpaceMemberID
	Title               *string
	Body                string
	BodyHTML            string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
