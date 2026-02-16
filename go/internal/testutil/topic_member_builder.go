package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicMemberBuilder はトピックメンバーテストデータのビルダー
type TopicMemberBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID       string
	topicID       string
	spaceMemberID string
	role          int32
	joinedAt      time.Time
}

// NewTopicMemberBuilder は TopicMemberBuilder を生成します
func NewTopicMemberBuilder(t *testing.T, tx *sql.Tx) *TopicMemberBuilder {
	t.Helper()
	return &TopicMemberBuilder{
		t:        t,
		tx:       tx,
		role:     0, // admin
		joinedAt: time.Now(),
	}
}

// WithSpaceID はスペースIDを設定します
func (b *TopicMemberBuilder) WithSpaceID(spaceID model.SpaceID) *TopicMemberBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithTopicID はトピックIDを設定します
func (b *TopicMemberBuilder) WithTopicID(topicID model.TopicID) *TopicMemberBuilder {
	b.topicID = string(topicID)
	return b
}

// WithSpaceMemberID はスペースメンバーIDを設定します
func (b *TopicMemberBuilder) WithSpaceMemberID(spaceMemberID model.SpaceMemberID) *TopicMemberBuilder {
	b.spaceMemberID = string(spaceMemberID)
	return b
}

// WithRole はロールを設定します
func (b *TopicMemberBuilder) WithRole(role int32) *TopicMemberBuilder {
	b.role = role
	return b
}

// Build はトピックメンバーを作成し、IDを返します
func (b *TopicMemberBuilder) Build() model.TopicMemberID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("TopicMemberBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.topicID == "" {
		b.t.Fatal("TopicMemberBuilder: topicIDが設定されていません。WithTopicID()を呼んでください")
	}
	if b.spaceMemberID == "" {
		b.t.Fatal("TopicMemberBuilder: spaceMemberIDが設定されていません。WithSpaceMemberID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO topic_members (space_id, topic_id, space_member_id, role, joined_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.spaceID, b.topicID, b.spaceMemberID, b.role, b.joinedAt, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("トピックメンバー作成に失敗: %v", err)
	}

	return model.TopicMemberID(id)
}
