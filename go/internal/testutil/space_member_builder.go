package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SpaceMemberBuilder はスペースメンバーテストデータのビルダー
type SpaceMemberBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID  string
	userID   model.UserID
	scopes   []string
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
		scopes:   []string{string(model.ScopeSpaceAdmin)},
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
func (b *SpaceMemberBuilder) WithUserID(userID model.UserID) *SpaceMemberBuilder {
	b.userID = userID
	return b
}

// WithActive はアクティブ状態を設定します
func (b *SpaceMemberBuilder) WithActive(active bool) *SpaceMemberBuilder {
	b.active = active
	return b
}

// WithScopes はスコープを設定します
func (b *SpaceMemberBuilder) WithScopes(scopes []model.Scope) *SpaceMemberBuilder {
	ss := make([]string, len(scopes))
	for i, s := range scopes {
		ss[i] = string(s)
	}
	b.scopes = ss
	return b
}

// WithJoinedAt は参加日時を設定します
func (b *SpaceMemberBuilder) WithJoinedAt(joinedAt time.Time) *SpaceMemberBuilder {
	b.joinedAt = joinedAt
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
		`INSERT INTO space_members (space_id, user_id, scopes, joined_at, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.spaceID, string(b.userID), pq.Array(b.scopes), b.joinedAt, b.active, now, now,
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
	userID   model.UserID
	scopes   []string
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
		scopes:   []string{string(model.ScopeSpaceAdmin)},
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
func (b *SpaceMemberBuilderDB) WithUserID(userID model.UserID) *SpaceMemberBuilderDB {
	b.userID = userID
	return b
}

// WithScopes はスコープを設定します
func (b *SpaceMemberBuilderDB) WithScopes(scopes []model.Scope) *SpaceMemberBuilderDB {
	ss := make([]string, len(scopes))
	for i, s := range scopes {
		ss[i] = string(s)
	}
	b.scopes = ss
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
		`INSERT INTO space_members (space_id, user_id, scopes, joined_at, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.spaceID, string(b.userID), pq.Array(b.scopes), b.joinedAt, b.active, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("スペースメンバー作成に失敗: %v", err)
	}

	return model.SpaceMemberID(id)
}
