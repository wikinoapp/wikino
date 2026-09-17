package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicBuilderはトピックテストデータのビルダー
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

// NewTopicBuilderはTopicBuilderを生成します
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

// WithSpaceIDはスペースIDを設定します
func (b *TopicBuilder) WithSpaceID(spaceID model.SpaceID) *TopicBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithNumberはナンバーを設定します
func (b *TopicBuilder) WithNumber(number int32) *TopicBuilder {
	b.number = number
	return b
}

// WithNameは名前を設定します
func (b *TopicBuilder) WithName(name string) *TopicBuilder {
	b.name = name
	return b
}

// WithDescriptionは説明を設定します
func (b *TopicBuilder) WithDescription(description string) *TopicBuilder {
	b.description = description
	return b
}

// WithVisibilityは公開範囲を設定します
func (b *TopicBuilder) WithVisibility(visibility int32) *TopicBuilder {
	b.visibility = visibility
	return b
}

// WithDiscardedは廃棄済み状態に設定します
func (b *TopicBuilder) WithDiscarded() *TopicBuilder {
	now := time.Now()
	b.discardedAt = &now
	return b
}

// Buildはトピックを作成し、IDを返します
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

// TopicBuilderDBはDBを直接使用するトピックテストデータのビルダー
// トランザクション管理を自前で行うUsecaseのテストに使用します
type TopicBuilderDB struct {
	t  *testing.T
	db *sql.DB

	spaceID     string
	number      int32
	name        string
	description string
	visibility  int32
}

// NewTopicBuilderDBはTopicBuilderDBを生成します
func NewTopicBuilderDB(t *testing.T, db *sql.DB) *TopicBuilderDB {
	t.Helper()
	return &TopicBuilderDB{
		t:           t,
		db:          db,
		number:      1,
		name:        "General",
		description: "",
		visibility:  0,
	}
}

// WithSpaceIDはスペースIDを設定します
func (b *TopicBuilderDB) WithSpaceID(spaceID model.SpaceID) *TopicBuilderDB {
	b.spaceID = string(spaceID)
	return b
}

// WithNumberはナンバーを設定します
func (b *TopicBuilderDB) WithNumber(number int32) *TopicBuilderDB {
	b.number = number
	return b
}

// WithNameは名前を設定します
func (b *TopicBuilderDB) WithName(name string) *TopicBuilderDB {
	b.name = name
	return b
}

// WithVisibilityは公開設定を設定します
func (b *TopicBuilderDB) WithVisibility(visibility int32) *TopicBuilderDB {
	b.visibility = visibility
	return b
}

// Buildはトピックを作成し、IDを返します
func (b *TopicBuilderDB) Build() model.TopicID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("TopicBuilderDB: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.db.QueryRowContext(
		context.Background(),
		`INSERT INTO topics (space_id, number, name, description, visibility, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.spaceID, b.number, b.name, b.description, b.visibility, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("トピック作成に失敗: %v", err)
	}

	return model.TopicID(id)
}
