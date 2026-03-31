package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// topicOwnerPolicy はスペースオーナー用のポリシー
// スペースオーナーは同じスペース内の全トピックのページを編集可能
type topicOwnerPolicy struct {
	spaceID           model.SpaceID
	spaceMemberActive bool
}

func (p *topicOwnerPolicy) CanCreatePage(topic *model.Topic) bool {
	return p.spaceMemberActive && p.spaceID == topic.Space.ID
}

func (p *topicOwnerPolicy) CanUpdatePage(page *model.Page) bool {
	return p.spaceMemberActive && p.spaceID == page.SpaceID
}

func (p *topicOwnerPolicy) CanUpdateDraftPage(draftPage *model.DraftPage) bool {
	return p.spaceMemberActive && p.spaceID == draftPage.SpaceID
}

func (p *topicOwnerPolicy) CanCreateSuggestion(topic *model.Topic) bool {
	return p.spaceMemberActive && p.spaceID == topic.Space.ID
}

func (p *topicOwnerPolicy) CanApplySuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.spaceID == suggestion.SpaceID
}

func (p *topicOwnerPolicy) CanCloseSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.spaceID == suggestion.SpaceID
}

func (p *topicOwnerPolicy) CanUpdateSuggestion(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.spaceID == suggestion.SpaceID && suggestion.Status == model.SuggestionStatusOpen
}

func (p *topicOwnerPolicy) CanEditSuggestionPage(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.spaceID == suggestion.SpaceID
}

func (p *topicOwnerPolicy) CanCreateSuggestionComment(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.spaceID == suggestion.SpaceID
}

func (p *topicOwnerPolicy) CanUpdateSuggestionComment(suggestion *model.Suggestion) bool {
	return p.spaceMemberActive && p.spaceID == suggestion.SpaceID && suggestion.Status == model.SuggestionStatusOpen
}
