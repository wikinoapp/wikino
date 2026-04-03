// Package policy はリソースに対する権限チェックを提供する
package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicPolicy はトピック内のリソースに対する権限を判定するインターフェース
type TopicPolicy interface {
	CanCreatePage(topic *model.Topic) bool
	CanUpdatePage(page *model.Page) bool
	CanUpdateDraftPage(draftPage *model.DraftPage) bool
	CanCreateSuggestion(topic *model.Topic) bool
	CanApplySuggestion(suggestion *model.Suggestion) bool
	CanCloseSuggestion(suggestion *model.Suggestion) bool
	CanUpdateSuggestion(suggestion *model.Suggestion) bool
	CanAddSuggestionPage(suggestion *model.Suggestion) bool
	CanRemoveSuggestionPage(suggestion *model.Suggestion) bool
	CanEditSuggestionPage(suggestion *model.Suggestion) bool
	CanCreateSuggestionComment(suggestion *model.Suggestion) bool
	CanUpdateSuggestionComment(suggestion *model.Suggestion) bool
}

// NewTopicPolicy はスペースメンバー・トピックメンバー情報から適切なポリシーを生成する
func NewTopicPolicy(spaceMember *model.SpaceMember, topicMember *model.TopicMember) TopicPolicy {
	if spaceMember.Role == model.SpaceMemberRoleOwner {
		return &topicOwnerPolicy{spaceID: spaceMember.SpaceID, spaceMemberActive: spaceMember.Active}
	}

	if topicMember == nil {
		return &topicGuestPolicy{spaceMemberID: spaceMember.ID, spaceMemberActive: spaceMember.Active}
	}

	if topicMember.Role == model.TopicMemberRoleAdmin {
		return &topicAdminPolicy{spaceMemberID: spaceMember.ID, topicID: topicMember.TopicID, spaceMemberActive: spaceMember.Active}
	}

	return &topicMemberPolicy{spaceMemberID: spaceMember.ID, topicID: topicMember.TopicID, spaceMemberActive: spaceMember.Active}
}
