package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// topicMemberPolicy はトピックMember用のポリシー
// トピックMemberは所属トピックのページを編集可能
type topicMemberPolicy struct {
	spaceMemberID     model.SpaceMemberID
	topicID           model.TopicID
	spaceMemberActive bool
}

func (p *topicMemberPolicy) CanCreatePage(topic *model.Topic) bool {
	return p.spaceMemberActive && p.topicID == topic.ID
}

func (p *topicMemberPolicy) CanUpdatePage(page *model.Page) bool {
	return p.spaceMemberActive && p.topicID == page.TopicID
}

func (p *topicMemberPolicy) CanUpdateDraftPage(draftPage *model.DraftPage) bool {
	return p.spaceMemberActive && p.topicID == draftPage.TopicID
}

func (p *topicMemberPolicy) CanApplySuggestion(_ *model.Suggestion) bool {
	return false
}

func (p *topicMemberPolicy) CanCloseSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.spaceMemberID == suggestion.CreatedSpaceMemberID
}

func (p *topicMemberPolicy) CanUpdateSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.topicID == suggestion.TopicID && suggestion.Status == model.SuggestionStatusOpen
}

func (p *topicMemberPolicy) CanUpdateSuggestionComment(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.topicID == suggestion.TopicID && suggestion.Status == model.SuggestionStatusOpen
}
