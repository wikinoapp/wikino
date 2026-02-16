package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// topicMemberPolicy はトピックMember用のポリシー
// トピックMemberは所属トピックのページを編集可能
type topicMemberPolicy struct {
	topicID model.TopicID
	active  bool
}

func (p *topicMemberPolicy) CanUpdatePage(page *model.Page) bool {
	return p.active && p.topicID == page.TopicID
}

func (p *topicMemberPolicy) CanUpdateDraftPage(draftPage *model.DraftPage) bool {
	return p.active && p.topicID == draftPage.TopicID
}
