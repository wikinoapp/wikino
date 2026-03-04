package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicForSidebar はサイドバーに表示するトピック情報です
type TopicForSidebar struct {
	Name            string
	Number          int32
	IconName        IconName
	SpaceIdentifier string
	SpaceName       string
}

// NewTopicsForSidebar はモデルのスライスからサイドバー用トピックのスライスを生成します
func NewTopicsForSidebar(topics []*model.Topic) []TopicForSidebar {
	result := make([]TopicForSidebar, len(topics))
	for i, t := range topics {
		result[i] = TopicForSidebar{
			Name:            t.Name,
			Number:          t.Number,
			IconName:        topicVisibilityIconName(t.Visibility),
			SpaceIdentifier: t.Space.Identifier.String(),
			SpaceName:       t.Space.Name,
		}
	}
	return result
}
