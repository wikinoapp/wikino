package viewmodel

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// DraftPageForSidebar はサイドバーに表示する下書きページ情報です
type DraftPageForSidebar struct {
	title           string
	PageNumber      int32
	TopicName       string
	IconName        IconName
	SpaceIdentifier string
}

// DisplayTitle は表示用タイトルを返します。タイトルが未設定の場合は「無題」を返します。
func (d DraftPageForSidebar) DisplayTitle(ctx context.Context) string {
	if d.title != "" {
		return d.title
	}
	return i18n.T(ctx, "sidebar_draft_pages_untitled")
}

// NewDraftPagesForSidebar はモデルのスライスからサイドバー用下書きページのスライスを生成します
func NewDraftPagesForSidebar(drafts []*model.DraftPage) []DraftPageForSidebar {
	result := make([]DraftPageForSidebar, len(drafts))
	for i, d := range drafts {
		result[i] = DraftPageForSidebar{
			title:           draftPageTitle(d),
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
