// Package policy はリソースに対する権限チェックを提供する
package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicPolicy はトピック内のリソースに対する権限を判定するインターフェース
type TopicPolicy interface {
	CanUpdatePage(page *model.Page) bool
	CanUpdateDraftPage(draftPage *model.DraftPage) bool
}

// NewTopicPolicy はスペースメンバー・トピックメンバー情報から適切なポリシーを生成する
func NewTopicPolicy(spaceMember *model.SpaceMember, topicMember *model.TopicMember) TopicPolicy {
	if spaceMember.Role == model.SpaceMemberRoleOwner {
		return &topicOwnerPolicy{spaceID: spaceMember.SpaceID, active: spaceMember.Active}
	}

	if topicMember == nil {
		return &topicGuestPolicy{}
	}

	if topicMember.Role == model.TopicMemberRoleAdmin {
		return &topicAdminPolicy{topicID: topicMember.TopicID, active: spaceMember.Active}
	}

	return &topicMemberPolicy{topicID: topicMember.TopicID, active: spaceMember.Active}
}
