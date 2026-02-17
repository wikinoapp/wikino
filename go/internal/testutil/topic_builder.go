package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicBuilder はトピックテストデータのビルダー
type TopicBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID     string
	number      int32
	name        string
	description string
	visibility  int32
	discardedAt *time.Time
}

// NewTopicBuilder は TopicBuilder を生成します
func NewTopicBuilder(t *testing.T, tx *sql.Tx) *TopicBuilder {
	t.Helper()
	return &TopicBuilder{
		t:           t,
		tx:          tx,
		number:      1,
		name:        "General",
		description: "",
		visibility:  0, // public
	}
}

// WithSpaceID はスペースIDを設定します
func (b *TopicBuilder) WithSpaceID(spaceID model.SpaceID) *TopicBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithNumber はナンバーを設定します
func (b *TopicBuilder) WithNumber(number int32) *TopicBuilder {
	b.number = number
	return b
}

// WithName は名前を設定します
func (b *TopicBuilder) WithName(name string) *TopicBuilder {
	b.name = name
	return b
}

// WithDescription は説明を設定します
func (b *TopicBuilder) WithDescription(description string) *TopicBuilder {
	b.description = description
	return b
}

// WithVisibility は公開範囲を設定します
func (b *TopicBuilder) WithVisibility(visibility int32) *TopicBuilder {
	b.visibility = visibility
	return b
}

// WithDiscarded は廃棄済み状態に設定します
func (b *TopicBuilder) WithDiscarded() *TopicBuilder {
	now := time.Now()
	b.discardedAt = &now
	return b
}

// Build はトピックを作成し、IDを返します
func (b *TopicBuilder) Build() model.TopicID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("TopicBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO topics (space_id, number, name, description, visibility, discarded_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		b.spaceID, b.number, b.name, b.description, b.visibility, b.discardedAt, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("トピック作成に失敗: %v", err)
	}

	return model.TopicID(id)
}
