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

// CardLinkTopic represents a reusable topic link card shared by the home page's "joined topics"
// section and the space page's topic section. It carries the topic name and number, the owning
// space's identifier (always set so path helpers can build links across multiple spaces) and
// name (rendered as a label only when non-empty), the topic visibility icon (public/private)
// shown as the card's leading icon, and whether the current user may create a page under the
// topic (used to toggle the new-page link).
//
// [Ja] CardLinkTopic はホーム画面の「参加中のトピック」セクションとスペース画面のトピック
// セクションで共有する、再利用可能なトピックリンクカード。トピック名・番号に加え、スペース
// 識別子 (複数スペースをまたぐリンクを組み立てられるよう常にセットする) と名前 (空でないときだけ
// ラベルとして表示する)、カード左側のリーディングアイコンとして表示するトピックの公開範囲アイコン
// (公開 / 非公開)、現在のユーザーがそのトピック配下でページを作成できるか (新規ページリンクの
// 表示切り替えに使う) を保持する。
type CardLinkTopic struct {
	Name            string
	Number          int32
	SpaceIdentifier SpaceIdentifier
	SpaceName       string
	TopicIconName   IconName
	CanCreatePage   bool
}

// NewCardLinkTopic builds a CardLinkTopic from the model and the create permission.
// [Ja] NewCardLinkTopic はモデルと作成権限から CardLinkTopic を生成する。
func NewCardLinkTopic(topic *model.Topic, canCreatePage bool) CardLinkTopic {
	return CardLinkTopic{
		Name:            topic.Name,
		Number:          topic.Number,
		SpaceIdentifier: NewSpaceIdentifier(topic.Space.Identifier),
		SpaceName:       topic.Space.Name,
		TopicIconName:   topicVisibilityIconName(topic.Visibility),
		CanCreatePage:   canCreatePage,
	}
}

// NewCardLinkTopics builds the cards from the topics and the per-topic create permission map.
// Topics missing from the map are treated as not creatable.
//
// [Ja] NewCardLinkTopics はトピックのスライスと、トピックごとの作成権限マップからカードの
// スライスを生成する。マップに無いトピックは作成権限なし扱いになる。
func NewCardLinkTopics(topics []*model.Topic, canCreatePageByTopic map[model.TopicID]bool) []CardLinkTopic {
	result := make([]CardLinkTopic, len(topics))
	for i, t := range topics {
		result[i] = NewCardLinkTopic(t, canCreatePageByTopic[t.ID])
	}
	return result
}
