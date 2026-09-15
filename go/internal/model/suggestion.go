package model

import (
	"time"
)

// SuggestionStatusは編集提案のステータスを表す
type SuggestionStatus int32

const (
	// SuggestionStatusDraftは下書きステータス
	SuggestionStatusDraft SuggestionStatus = 0
	// SuggestionStatusOpenはオープンステータス
	SuggestionStatusOpen SuggestionStatus = 1
	// SuggestionStatusAppliedは反映済みステータス
	SuggestionStatusApplied SuggestionStatus = 2
	// SuggestionStatusClosedはクローズステータス
	SuggestionStatusClosed SuggestionStatus = 3
)

// Suggestionは編集提案のドメインモデル
type Suggestion struct {
	ID                   SuggestionID
	SpaceID              SpaceID
	TopicID              TopicID
	CreatedSpaceMemberID SpaceMemberID
	Number               SuggestionNumber
	Title                string
	Body                 string
	Status               SuggestionStatus
	AppliedAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
