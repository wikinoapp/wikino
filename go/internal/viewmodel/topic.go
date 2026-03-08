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

// TopicForSelect はセレクトボックス用のトピック情報です
type TopicForSelect struct {
	Name   string
	Number int32
}

// NewTopicForSelect はモデルからTopicForSelectを生成します
func NewTopicForSelect(topic *model.Topic) TopicForSelect {
	return TopicForSelect{
		Name:   topic.Name,
		Number: topic.Number,
	}
}

// TopicForShow はトピック詳細画面用のトピック情報です
type TopicForShow struct {
	Name          string
	Number        int32
	Description   string
	IconName      IconName
	CanUpdate     bool
	CanCreatePage bool
}

// NewTopicForShow はモデルからTopicForShowを生成します
func NewTopicForShow(topic *model.Topic, canUpdate bool, canCreatePage bool) TopicForShow {
	return TopicForShow{
		Name:          topic.Name,
		Number:        topic.Number,
		Description:   topic.Description,
		IconName:      topicVisibilityIconName(topic.Visibility),
		CanUpdate:     canUpdate,
		CanCreatePage: canCreatePage,
	}
}

// topicVisibilityIconName はトピックの公開範囲に対応するアイコン名を返します
func topicVisibilityIconName(v model.TopicVisibility) IconName {
	if v == model.TopicVisibilityPublic {
		return "globe-regular"
	}
	return "lock-regular"
}
