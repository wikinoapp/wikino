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

// TopicVisibilityIcon はトピックの公開範囲に対応するアイコン名を返します
func TopicVisibilityIcon(v TopicVisibility) string {
	if v == TopicVisibilityPublic {
		return "globe-regular"
	}
	return "lock-regular"
}

// Topic はトピックのドメインモデル
type Topic struct {
	ID          TopicID
	SpaceID     SpaceID
	Number      int32
	Name        string
	Description string
	Visibility  TopicVisibility
	DiscardedAt *time.Time
}
