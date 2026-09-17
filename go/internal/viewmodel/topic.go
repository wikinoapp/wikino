package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// Topicはテンプレートで表示するトピック情報です
type Topic struct {
	Name     string
	Number   int32
	IconName IconName
}

// NewTopicはモデルからTopicを生成します
func NewTopic(topic *model.Topic) Topic {
	return Topic{
		Name:     topic.Name,
		Number:   topic.Number,
		IconName: topicVisibilityIconName(topic.Visibility),
	}
}

// TopicForSelectはセレクトボックス用のトピック情報です
type TopicForSelect struct {
	Name   string
	Number int32
}

// NewTopicForSelectはモデルからTopicForSelectを生成します
func NewTopicForSelect(topic *model.Topic) TopicForSelect {
	return TopicForSelect{
		Name:   topic.Name,
		Number: topic.Number,
	}
}

// TopicForShowはトピック詳細画面用のトピック情報です
type TopicForShow struct {
	Name        string
	Number      int32
	Description string
	IconName    IconName
	// IsPublicは、画面がタイトルの下で説明文と同じ行に言葉で示す公開範囲を運ぶ。
	// アイコンだけでは状態を錠前から推し量ることになり、読み手によって受け取り方が変わるため、
	// ラベルはIconNameではなくこの値から選ぶ。
	IsPublic      bool
	CanUpdate     bool
	CanCreatePage bool
}

// NewTopicForShowはモデルからTopicForShowを生成します
func NewTopicForShow(topic *model.Topic, canUpdate bool, canCreatePage bool) TopicForShow {
	return TopicForShow{
		Name:          topic.Name,
		Number:        topic.Number,
		Description:   topic.Description,
		IconName:      topicVisibilityIconName(topic.Visibility),
		IsPublic:      topic.Visibility == model.TopicVisibilityPublic,
		CanUpdate:     canUpdate,
		CanCreatePage: canCreatePage,
	}
}

// topicVisibilityIconNameはトピックの公開範囲に対応するアイコン名を返します
func topicVisibilityIconName(v model.TopicVisibility) IconName {
	if v == model.TopicVisibilityPublic {
		return "globe-regular"
	}
	return "lock-regular"
}

// CardLinkTopicはホーム画面の「参加中のトピック」セクションとスペース画面のトピック
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

// NewCardLinkTopicはモデルと作成権限からCardLinkTopicを生成する。
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

// NewCardLinkTopicsはトピックのスライスと、トピックごとの作成権限マップからカードの
// スライスを生成する。マップに無いトピックは作成権限なし扱いになる。
func NewCardLinkTopics(topics []*model.Topic, canCreatePageByTopic map[model.TopicID]bool) []CardLinkTopic {
	result := make([]CardLinkTopic, len(topics))
	for i, t := range topics {
		result[i] = NewCardLinkTopic(t, canCreatePageByTopic[t.ID])
	}
	return result
}

// NewCardLinkTopicsForSpaceはスペース詳細のトピックセクション用にカードを生成する。
// 対象トピックはすべてspaceIdentifierで識別される単一スペースに属する。これらのトピックは
// SpaceIDしか持たない (Spaceは読み込まれていない) ため、スペース識別子は明示的に渡す。
// スペース名はヘッダーで既に表示しているため、SpaceNameを空にしてカード内のスペース名表示を抑止する。
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
