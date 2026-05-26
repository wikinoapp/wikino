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

// TopicForSpaceSection is a topic shown in the space detail's topic section. Each topic links to
// its detail and, when CanCreatePage is true, offers a per-topic "new page" action. Description and
// update permission are intentionally omitted since the section only navigates and creates pages.
//
// [Ja] TopicForSpaceSection はスペース詳細のトピックセクションに表示するトピック。各トピックは
// 詳細へリンクし、CanCreatePage が true のときトピックごとの「新規ページ」アクションを出す。
// セクションは遷移とページ作成だけを担うため、説明文や更新権限は意図的に持たせない。
type TopicForSpaceSection struct {
	Name          string
	Number        int32
	IconName      IconName
	CanCreatePage bool
}

// NewTopicForSpaceSection builds a TopicForSpaceSection from the model and the create permission.
// [Ja] NewTopicForSpaceSection はモデルと作成権限から TopicForSpaceSection を生成します。
func NewTopicForSpaceSection(topic *model.Topic, canCreatePage bool) TopicForSpaceSection {
	return TopicForSpaceSection{
		Name:          topic.Name,
		Number:        topic.Number,
		IconName:      topicVisibilityIconName(topic.Visibility),
		CanCreatePage: canCreatePage,
	}
}

// NewTopicsForSpaceSection builds the section topics from the topics and the per-topic create
// permission map. Topics missing from the map are treated as not creatable.
//
// [Ja] NewTopicsForSpaceSection はトピックのスライスと、トピックごとの作成権限マップから
// TopicForSpaceSection のスライスを生成する。マップに無いトピックは作成権限なし扱いになる。
func NewTopicsForSpaceSection(topics []*model.Topic, canCreatePageByTopic map[model.TopicID]bool) []TopicForSpaceSection {
	result := make([]TopicForSpaceSection, len(topics))
	for i, topic := range topics {
		result[i] = NewTopicForSpaceSection(topic, canCreatePageByTopic[topic.ID])
	}
	return result
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
// (used to render the per-card SpaceIcon and the space label), and the topic visibility
// icon (public/private) shown as the card's leading icon.
//
// [Ja] JoinedTopicCard はホーム画面の「参加中のトピック」セクションに表示するトピックカード。
// トピック名・番号に加え、カード内の SpaceIcon とスペース名表示のためのスペース識別子と名前、
// カード左側のリーディングアイコンとして表示するトピックの公開範囲アイコン (公開 / 非公開) を保持する。
type JoinedTopicCard struct {
	Name            string
	Number          int32
	SpaceIdentifier SpaceIdentifier
	SpaceName       string
	TopicIconName   IconName
}

// NewJoinedTopicCard はモデルからホーム画面用 JoinedTopicCard を生成する
func NewJoinedTopicCard(topic *model.Topic) JoinedTopicCard {
	return JoinedTopicCard{
		Name:            topic.Name,
		Number:          topic.Number,
		SpaceIdentifier: NewSpaceIdentifier(topic.Space.Identifier),
		SpaceName:       topic.Space.Name,
		TopicIconName:   topicVisibilityIconName(topic.Visibility),
	}
}

// NewJoinedTopicCards はモデルのスライスからホーム画面用 JoinedTopicCard のスライスを生成する
func NewJoinedTopicCards(topics []*model.Topic) []JoinedTopicCard {
	result := make([]JoinedTopicCard, len(topics))
	for i, t := range topics {
		result[i] = NewJoinedTopicCard(t)
	}
	return result
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
