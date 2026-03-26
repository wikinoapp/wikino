package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// topicAdminPolicy はトピックAdmin用のポリシー
// トピックAdminは所属トピックのページを編集可能
type topicAdminPolicy struct {
	spaceMemberID     model.SpaceMemberID
	topicID           model.TopicID
	spaceMemberActive bool
}

func (p *topicAdminPolicy) CanCreatePage(topic *model.Topic) bool {
	return p.spaceMemberActive && p.topicID == topic.ID
}

func (p *topicAdminPolicy) CanUpdatePage(page *model.Page) bool {
	return p.spaceMemberActive && p.topicID == page.TopicID
}

func (p *topicAdminPolicy) CanUpdateDraftPage(draftPage *model.DraftPage) bool {
	return p.spaceMemberActive && p.topicID == draftPage.TopicID
}

func (p *topicAdminPolicy) CanApplySuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.topicID == suggestion.TopicID
}

func (p *topicAdminPolicy) CanCloseSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && (p.spaceMemberID == suggestion.CreatedSpaceMemberID || p.topicID == suggestion.TopicID)
}

func (p *topicAdminPolicy) CanUpdateSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.topicID == suggestion.TopicID && suggestion.Status == model.SuggestionStatusOpen
}

func (p *topicAdminPolicy) CanUpdateSuggestionComment(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.topicID == suggestion.TopicID && suggestion.Status == model.SuggestionStatusOpen
}
