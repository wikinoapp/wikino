package model

import (
	"time"
)

// SuggestionStatus は編集提案のステータスを表す
type SuggestionStatus int32

const (
	// SuggestionStatusDraft は下書きステータス
	SuggestionStatusDraft SuggestionStatus = 0
	// SuggestionStatusOpen はオープンステータス
	SuggestionStatusOpen SuggestionStatus = 1
	// SuggestionStatusApplied は反映済みステータス
	SuggestionStatusApplied SuggestionStatus = 2
	// SuggestionStatusClosed はクローズステータス
	SuggestionStatusClosed SuggestionStatus = 3
)

// Suggestion は編集提案のドメインモデル
type Suggestion struct {
	ID                   SuggestionID
	SpaceID              SpaceID
	TopicID              TopicID
	CreatedSpaceMemberID SpaceMemberID
	Title                string
	Body                 string
	BodyHTML             string
	Status               SuggestionStatus
	AppliedAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
