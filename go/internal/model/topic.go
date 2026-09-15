package model

import (
	"time"
)

// TopicVisibilityはトピックの公開範囲を表す
type TopicVisibility int32

const (
	// TopicVisibilityPublicは公開トピック
	TopicVisibilityPublic TopicVisibility = 0
	// TopicVisibilityPrivateは非公開トピック
	TopicVisibilityPrivate TopicVisibility = 1
)

// Topicはトピックのドメインモデル
type Topic struct {
	ID          TopicID
	Space       *Space
	Number      int32
	Name        string
	Description string
	Visibility  TopicVisibility
	DiscardedAt *time.Time
}

// トピックのフォームが送信する公開範囲の値。
const (
	topicVisibilityPublicValue  = "public"
	topicVisibilityPrivateValue = "private"
)

// ParseTopicVisibilityはトピックのフォームが送信する値をTopicVisibilityに変換する。
// 2つ目の戻り値はその値がフォームの提示する2つのいずれかであるかを返す。それ以外の値を、
// 送信者が選んでいない公開範囲へ倒さずに拒否できるようにするためである。
func ParseTopicVisibility(value string) (TopicVisibility, bool) {
	switch value {
	case topicVisibilityPublicValue:
		return TopicVisibilityPublic, true
	case topicVisibilityPrivateValue:
		return TopicVisibilityPrivate, true
	default:
		return TopicVisibilityPublic, false
	}
}

// Stringはトピックのフォームが公開範囲として送信する値を返す。
func (v TopicVisibility) String() string {
	if v == TopicVisibilityPrivate {
		return topicVisibilityPrivateValue
	}
	return topicVisibilityPublicValue
}
