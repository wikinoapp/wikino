package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SpaceMemberBuilder はスペースメンバーテストデータのビルダー
type SpaceMemberBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID  string
	userID   string
	role     int32
	joinedAt time.Time
	active   bool
}

// NewSpaceMemberBuilder は SpaceMemberBuilder を生成します
func NewSpaceMemberBuilder(t *testing.T, tx *sql.Tx) *SpaceMemberBuilder {
	t.Helper()
	now := time.Now()
	return &SpaceMemberBuilder{
		t:        t,
		tx:       tx,
		role:     0, // owner
		joinedAt: now,
		active:   true,
	}
}

// WithSpaceID はスペースIDを設定します
func (b *SpaceMemberBuilder) WithSpaceID(spaceID model.SpaceID) *SpaceMemberBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithUserID はユーザーIDを設定します
func (b *SpaceMemberBuilder) WithUserID(userID string) *SpaceMemberBuilder {
	b.userID = userID
	return b
}

// WithRole はロールを設定します
func (b *SpaceMemberBuilder) WithRole(role int32) *SpaceMemberBuilder {
	b.role = role
	return b
}

// WithActive はアクティブ状態を設定します
func (b *SpaceMemberBuilder) WithActive(active bool) *SpaceMemberBuilder {
	b.active = active
	return b
}

// Build はスペースメンバーを作成し、IDを返します
func (b *SpaceMemberBuilder) Build() model.SpaceMemberID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("SpaceMemberBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.userID == "" {
		b.t.Fatal("SpaceMemberBuilder: userIDが設定されていません。WithUserID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO space_members (space_id, user_id, role, joined_at, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.spaceID, b.userID, b.role, b.joinedAt, b.active, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("スペースメンバー作成に失敗: %v", err)
	}

	return model.SpaceMemberID(id)
}

// SpaceMemberBuilderDB はDBを直接使用するスペースメンバーテストデータのビルダー
// トランザクション管理を自前で行うUsecaseのテストに使用します
type SpaceMemberBuilderDB struct {
	t  *testing.T
	db *sql.DB

	spaceID  string
	userID   string
	role     int32
	joinedAt time.Time
	active   bool
}

// NewSpaceMemberBuilderDB は SpaceMemberBuilderDB を生成します
func NewSpaceMemberBuilderDB(t *testing.T, db *sql.DB) *SpaceMemberBuilderDB {
	t.Helper()
	now := time.Now()
	return &SpaceMemberBuilderDB{
		t:        t,
		db:       db,
		role:     0,
		joinedAt: now,
		active:   true,
	}
}

// WithSpaceID はスペースIDを設定します
func (b *SpaceMemberBuilderDB) WithSpaceID(spaceID model.SpaceID) *SpaceMemberBuilderDB {
	b.spaceID = string(spaceID)
	return b
}

// WithUserID はユーザーIDを設定します
func (b *SpaceMemberBuilderDB) WithUserID(userID string) *SpaceMemberBuilderDB {
	b.userID = userID
	return b
}

// Build はスペースメンバーを作成し、IDを返します
func (b *SpaceMemberBuilderDB) Build() model.SpaceMemberID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("SpaceMemberBuilderDB: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.userID == "" {
		b.t.Fatal("SpaceMemberBuilderDB: userIDが設定されていません。WithUserID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.db.QueryRowContext(
		context.Background(),
		`INSERT INTO space_members (space_id, user_id, role, joined_at, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.spaceID, b.userID, b.role, b.joinedAt, b.active, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("スペースメンバー作成に失敗: %v", err)
	}

	return model.SpaceMemberID(id)
}
