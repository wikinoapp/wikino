package model

import (
	"time"
)

// SuggestionComment は編集提案コメントのドメインモデル
type SuggestionComment struct {
	ID                   SuggestionCommentID
	SpaceID              SpaceID
	SuggestionID         SuggestionID
	CreatedSpaceMemberID SpaceMemberID
	Number               SuggestionCommentNumber
	Body                 string
	BodyHTML             string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
