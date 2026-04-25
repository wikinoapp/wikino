package viewmodel

import (
	"context"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// DraftPageForIndex は下書き一覧画面に表示する下書きページ情報です
type DraftPageForIndex struct {
	title           string
	PageNumber      int32
	SpaceIdentifier string
	ModifiedAt      time.Time
}

// DisplayTitle は表示用タイトルを返します。タイトルが未設定の場合は「無題」を返します。
func (d DraftPageForIndex) DisplayTitle(ctx context.Context) string {
	if d.title != "" {
		return d.title
	}
	return i18n.T(ctx, "draft_page_index_untitled")
}

// DraftPageGroupForIndex は下書き一覧画面のスペース・トピック単位のグループです
type DraftPageGroupForIndex struct {
	SpaceName       string
	SpaceIdentifier string
	TopicName       string
	TopicNumber     int32
	TopicIconName   IconName
	DraftPages      []DraftPageForIndex
}

// NewDraftPageGroupsForIndex はモデルのスライスからスペース・トピック単位のグループに変換します。
// モデルはスペース名・トピック名順にソート済みの前提です。
func NewDraftPageGroupsForIndex(drafts []*model.DraftPage) []DraftPageGroupForIndex {
	if len(drafts) == 0 {
		return nil
	}

	var groups []DraftPageGroupForIndex
	var current *DraftPageGroupForIndex

	for _, d := range drafts {
		spaceIdentifier := d.Topic.Space.Identifier.String()
		topicName := d.Topic.Name
		spaceName := d.Topic.Space.Name

		if current == nil || current.SpaceIdentifier != spaceIdentifier || current.TopicName != topicName {
			if current != nil {
				groups = append(groups, *current)
			}
			current = &DraftPageGroupForIndex{
				SpaceName:       spaceName,
				SpaceIdentifier: spaceIdentifier,
				TopicName:       topicName,
				TopicNumber:     d.Topic.Number,
				TopicIconName:   topicVisibilityIconName(d.Topic.Visibility),
			}
		}

		current.DraftPages = append(current.DraftPages, DraftPageForIndex{
			title:           draftPageTitle(d),
			PageNumber:      int32(d.Page.Number),
			SpaceIdentifier: spaceIdentifier,
			ModifiedAt:      d.ModifiedAt,
		})
	}

	if current != nil {
		groups = append(groups, *current)
	}

	return groups
}
