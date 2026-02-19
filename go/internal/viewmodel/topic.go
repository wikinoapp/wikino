package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// Topic はテンプレートで表示するトピック情報です
type Topic struct {
	Name     string
	Number   int32
	IconName IconName
}

// NewTopic はモデルからTopicを生成します
func NewTopic(topic *model.Topic) Topic {
	return Topic{
		Name:     topic.Name,
		Number:   topic.Number,
		IconName: topicVisibilityIconName(topic.Visibility),
	}
}

// topicVisibilityIconName はトピックの公開範囲に対応するアイコン名を返します
func topicVisibilityIconName(v model.TopicVisibility) IconName {
	if v == model.TopicVisibilityPublic {
		return "globe-regular"
	}
	return "lock-regular"
}
