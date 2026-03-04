package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// topicMemberPolicy はトピックMember用のポリシー
// トピックMemberは所属トピックのページを編集可能
type topicMemberPolicy struct {
	topicID           model.TopicID
	spaceMemberActive bool
}

func (p *topicMemberPolicy) CanUpdatePage(page *model.Page) bool {
	return p.spaceMemberActive && p.topicID == page.TopicID
}

func (p *topicMemberPolicy) CanUpdateDraftPage(draftPage *model.DraftPage) bool {
	return p.spaceMemberActive && p.topicID == draftPage.TopicID
}
