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

func (p *topicGuestPolicy) CanCreatePage(_ *model.Topic) bool          { return false }
func (p *topicGuestPolicy) CanUpdatePage(_ *model.Page) bool           { return false }
func (p *topicGuestPolicy) CanUpdateDraftPage(_ *model.DraftPage) bool { return false }
func (p *topicGuestPolicy) CanCreateSuggestion(topic *model.Topic) bool {
	// 非トピックメンバーでも公開トピックには編集提案を作成可能
	return p.spaceMemberActive && topic.Visibility != model.TopicVisibilityPrivate
}

func (p *topicGuestPolicy) CanApplySuggestion(_ *model.Suggestion) bool { return false }

func (p *topicGuestPolicy) CanCloseSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.spaceMemberID == suggestion.CreatedSpaceMemberID
}

func (p *topicGuestPolicy) CanUpdateSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && suggestion.Status == model.SuggestionStatusOpen
}

func (p *topicGuestPolicy) CanAddSuggestionPage(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && suggestion.Status == model.SuggestionStatusOpen
}

func (p *topicGuestPolicy) CanRemoveSuggestionPage(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && suggestion.Status == model.SuggestionStatusOpen
}

func (p *topicGuestPolicy) CanEditSuggestionPage(_ *model.Suggestion) bool {
	return p.spaceMemberActive
}

func (p *topicGuestPolicy) CanCreateSuggestionComment(_ *model.Suggestion) bool {
	return p.spaceMemberActive
}

func (p *topicGuestPolicy) CanUpdateSuggestionComment(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && suggestion.Status == model.SuggestionStatusOpen
}
