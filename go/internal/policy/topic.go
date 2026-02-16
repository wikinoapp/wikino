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

// NewTopicPolicy はスペースメンバーのロールとトピックメンバー情報から適切なポリシーを生成する
// spaceMemberActive はスペースメンバーが有効かどうかを示す（SpaceMember.Active に対応）
func NewTopicPolicy(spaceMemberRole model.SpaceMemberRole, spaceID model.SpaceID, topicMember *model.TopicMember, spaceMemberActive bool) TopicPolicy {
	if spaceMemberRole == model.SpaceMemberRoleOwner {
		return &topicOwnerPolicy{spaceID: spaceID, active: spaceMemberActive}
	}

	if topicMember == nil {
		return &topicGuestPolicy{}
	}

	if topicMember.Role == model.TopicMemberRoleAdmin {
		return &topicAdminPolicy{topicID: topicMember.TopicID, active: spaceMemberActive}
	}

	return &topicMemberPolicy{topicID: topicMember.TopicID, active: spaceMemberActive}
}
