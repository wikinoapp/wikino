package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// DraftPageForSidebar はサイドバーに表示する下書きページ情報です
type DraftPageForSidebar struct {
	Title           string
	PageNumber      int32
	TopicName       string
	IconName        IconName
	SpaceIdentifier string
}

// NewDraftPagesForSidebar はモデルのスライスからサイドバー用下書きページのスライスを生成します
func NewDraftPagesForSidebar(drafts []*model.DraftPage) []DraftPageForSidebar {
	result := make([]DraftPageForSidebar, len(drafts))
	for i, d := range drafts {
		result[i] = DraftPageForSidebar{
			Title:           draftPageTitle(d),
			PageNumber:      int32(d.Page.Number),
			TopicName:       d.Topic.Name,
			IconName:        topicVisibilityIconName(d.Topic.Visibility),
			SpaceIdentifier: d.Topic.Space.Identifier.String(),
		}
	}
	return result
}

// draftPageTitle は下書きページの表示タイトルを返します
func draftPageTitle(d *model.DraftPage) string {
	if d.Title != nil && *d.Title != "" {
		return *d.Title
	}
	if d.Page != nil && d.Page.Title != nil && *d.Page.Title != "" {
		return *d.Page.Title
	}
	return ""
}
