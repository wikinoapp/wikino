package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// topicGuestPolicy は非トピックメンバー用のポリシー
// トピックに所属していないスペースメンバーはページを編集できない
type topicGuestPolicy struct {
	spaceMemberID     model.SpaceMemberID
	spaceMemberActive bool
}

func (p *topicGuestPolicy) CanCreatePage(_ *model.Topic) bool           { return false }
func (p *topicGuestPolicy) CanUpdatePage(_ *model.Page) bool            { return false }
func (p *topicGuestPolicy) CanUpdateDraftPage(_ *model.DraftPage) bool  { return false }
func (p *topicGuestPolicy) CanApplySuggestion(_ *model.Suggestion) bool { return false }

func (p *topicGuestPolicy) CanCloseSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.spaceMemberID == suggestion.CreatedSpaceMemberID
}

func (p *topicGuestPolicy) CanUpdateSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && suggestion.Status == model.SuggestionStatusOpen
}

func (p *topicGuestPolicy) CanUpdateSuggestionComment(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && suggestion.Status == model.SuggestionStatusOpen
}
