package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// topicGuestPolicy は非トピックメンバー用のポリシー
// トピックに所属していないスペースメンバーはページを編集できない
type topicGuestPolicy struct{}

func (p *topicGuestPolicy) CanCreatePage(_ *model.Topic) bool          { return false }
func (p *topicGuestPolicy) CanUpdatePage(_ *model.Page) bool           { return false }
func (p *topicGuestPolicy) CanUpdateDraftPage(_ *model.DraftPage) bool { return false }
