package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// topicAdminPolicy はトピックAdmin用のポリシー
// トピックAdminは所属トピックのページを編集可能
type topicAdminPolicy struct {
	topicID model.TopicID
	active  bool
}

func (p *topicAdminPolicy) CanUpdatePage(page *model.Page) bool {
	return p.active && p.topicID == page.TopicID
}

func (p *topicAdminPolicy) CanUpdateDraftPage(draftPage *model.DraftPage) bool {
	return p.active && p.topicID == draftPage.TopicID
}
