package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// CardLinkPage はリンク一覧・バックリンク一覧で使用するページカードの表示データです
type CardLinkPage struct {
	Title        string
	Number       int32
	TopicName    string
	TopicIcon    IconName
	Pinned       bool
	CardImageURL string
}

// NewCardLinkPage はmodel.Pageとトピック情報からカード用のビューモデルを生成します
func NewCardLinkPage(pg *model.Page, topicMap map[model.TopicID]*model.Topic) CardLinkPage {
	var title string
	if pg.Title != nil {
		title = *pg.Title
	}

	var topicName string
	var topicIcon IconName
	if topic, ok := topicMap[pg.TopicID]; ok {
		topicName = topic.Name
		topicIcon = topicVisibilityIconName(topic.Visibility)
	}

	return CardLinkPage{
		Title:     title,
		Number:    int32(pg.Number),
		TopicName: topicName,
		TopicIcon: topicIcon,
		Pinned:    pg.PinnedAt != nil,
	}
}
