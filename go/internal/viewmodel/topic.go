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

// NewCardLinkTopicsForSpace builds the cards for the space detail's topic section. All topics belong
// to the single space identified by spaceIdentifier, which is passed in explicitly because these
// topics carry only SpaceID (their Space is not loaded). SpaceName is left empty so the card hides
// the in-card space label, since the space header already shows the name. Topics missing from the
// map are treated as not creatable.
//
// [Ja] NewCardLinkTopicsForSpace はスペース詳細のトピックセクション用にカードを生成する。
// 対象トピックはすべて spaceIdentifier で識別される単一スペースに属する。これらのトピックは
// SpaceID しか持たない (Space は読み込まれていない) ため、スペース識別子は明示的に渡す。
// スペース名はヘッダーで既に表示しているため、SpaceName を空にしてカード内のスペース名表示を抑止する。
// マップに無いトピックは作成権限なし扱いになる。
func NewCardLinkTopicsForSpace(topics []*model.Topic, canCreatePageByTopic map[model.TopicID]bool, spaceIdentifier SpaceIdentifier) []CardLinkTopic {
	result := make([]CardLinkTopic, len(topics))
	for i, t := range topics {
		result[i] = CardLinkTopic{
			Name:            t.Name,
			Number:          t.Number,
			SpaceIdentifier: spaceIdentifier,
			SpaceName:       "",
			TopicIconName:   topicVisibilityIconName(t.Visibility),
			CanCreatePage:   canCreatePageByTopic[t.ID],
		}
	}
	return result
}
