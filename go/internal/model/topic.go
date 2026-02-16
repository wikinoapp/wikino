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
	SpaceID     SpaceID
	Number      int32
	Name        string
	Description string
	Visibility  TopicVisibility
	DiscardedAt *time.Time
}
