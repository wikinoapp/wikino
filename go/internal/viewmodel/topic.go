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

// JoinedTopicCard represents a topic card shown in the home page's "joined topics" section.
// It carries the topic name and number along with the owning space's identifier and name
// (used to render the per-card SpaceIcon and the space label) plus the published page count.
//
// [Ja] JoinedTopicCard はホーム画面の「参加中のトピック」セクションに表示するトピックカード。
// トピック名・番号に加え、カード内の SpaceIcon とスペース名表示のために
// スペース識別子と名前、公開中ページ数を保持する。
type JoinedTopicCard struct {
	Name                string
	Number              int32
	SpaceIdentifier     SpaceIdentifier
	SpaceName           string
	PublishedPagesCount int32
}

// NewJoinedTopicCard builds a JoinedTopicCard from a topic and its published page count.
// Repository converts query stats to (Topic, PublishedPagesCount) pairs; the caller iterates
// over those pairs and feeds each one to this constructor, keeping ViewModel free of any
// dependency on the Infrastructure layer.
//
// [Ja] NewJoinedTopicCard は model.Topic と公開中ページ数から JoinedTopicCard を生成する。
// Repository 戻り値の (Topic, PublishedPagesCount) ペアを 1 件ずつ受け取る形にすることで、
// ViewModel が Infrastructure 層 (repository パッケージ) に依存しないように分離している。
func NewJoinedTopicCard(topic *model.Topic, publishedPagesCount int32) JoinedTopicCard {
	return JoinedTopicCard{
		Name:                topic.Name,
		Number:              topic.Number,
		SpaceIdentifier:     NewSpaceIdentifier(topic.Space.Identifier),
		SpaceName:           topic.Space.Name,
		PublishedPagesCount: publishedPagesCount,
	}
}

// SpaceForIcon returns a Space view-model populated with the fields required to render
// the per-card SpaceIcon (identifier drives the deterministic background color and first
// character; name is kept available for future use).
//
// [Ja] SpaceForIcon はカード内の SpaceIcon を描画するために必要な Space ビューモデルを返す。
// identifier は背景色と頭文字の決定に使われ、name は将来の利用に備えて保持する。
func (c JoinedTopicCard) SpaceForIcon() Space {
	return Space{
		Name:       c.SpaceName,
		Identifier: c.SpaceIdentifier,
	}
}
