package viewmodel

import (
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SuggestionForList は編集提案一覧用の表示データです
type SuggestionForList struct {
	Number      int32
	Title       string
	Status      model.SuggestionStatus
	CreatorName string
	CreatedAt   time.Time
}

// NewSuggestionForListInput はNewSuggestionForListの入力パラメータです
type NewSuggestionForListInput struct {
	Suggestions []*model.Suggestion
	UserMap     map[model.SpaceMemberID]*model.User
}

// NewSuggestionsForList は編集提案モデルのスライスから一覧用ViewModelのスライスを生成します
func NewSuggestionsForList(input NewSuggestionForListInput) []SuggestionForList {
	items := make([]SuggestionForList, len(input.Suggestions))
	for i, s := range input.Suggestions {
		var creatorName string
		if user, ok := input.UserMap[s.CreatedSpaceMemberID]; ok {
			creatorName = userDisplayName(user)
		}
		items[i] = SuggestionForList{
			Number:      int32(s.Number),
			Title:       s.Title,
			Status:      s.Status,
			CreatorName: creatorName,
			CreatedAt:   s.CreatedAt,
		}
	}
	return items
}

// userDisplayName はユーザーの表示名を返す（名前があれば名前、なければアットネーム）
func userDisplayName(user *model.User) string {
	if user.Name != "" {
		return user.Name
	}
	return user.Atname
}
