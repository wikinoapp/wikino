package model

import (
	"time"
)

// TopicVisibility はトピックの公開範囲を表す
type TopicVisibility int32

const (
	// TopicVisibilityPublic は公開トピック
	TopicVisibilityPublic TopicVisibility = 0
	// TopicVisibilityPrivate は非公開トピック
	TopicVisibilityPrivate TopicVisibility = 1
)

// Topic はトピックのドメインモデル
type Topic struct {
	ID          TopicID
	Space       *Space
	Number      int32
	Name        string
	Description string
	Visibility  TopicVisibility
	DiscardedAt *time.Time
}

// Topic visibility as the topic form submits it.
//
// [Ja] トピックのフォームが送信する公開範囲の値。
const (
	topicVisibilityPublicValue  = "public"
	topicVisibilityPrivateValue = "private"
)

// ParseTopicVisibility turns the value a topic form submits into a TopicVisibility. The second
// return value reports whether the value is one of the two the form offers, so that anything else
// is refused rather than falling back to a visibility the submitter did not choose.
//
// [Ja] ParseTopicVisibility はトピックのフォームが送信する値を TopicVisibility に変換する。
// 2 つ目の戻り値はその値がフォームの提示する 2 つのいずれかであるかを返す。それ以外の値を、
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

// String returns the value a topic form submits for the visibility.
//
// [Ja] String はトピックのフォームが公開範囲として送信する値を返す。
func (v TopicVisibility) String() string {
	if v == TopicVisibilityPrivate {
		return topicVisibilityPrivateValue
	}
	return topicVisibilityPublicValue
}
