package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// topicOwnerPolicy はスペースオーナー用のポリシー
// スペースオーナーは同じスペース内の全トピックのページを編集可能
type topicOwnerPolicy struct {
	spaceID model.SpaceID
	active  bool
}

func (p *topicOwnerPolicy) CanUpdatePage(page *model.Page) bool {
	return p.active && p.spaceID == page.SpaceID
}

func (p *topicOwnerPolicy) CanUpdateDraftPage(draftPage *model.DraftPage) bool {
	return p.active && p.spaceID == draftPage.SpaceID
}
