package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SpaceBuilder はスペーステストデータのビルダー
type SpaceBuilder struct {
	t  *testing.T
	tx *sql.Tx

	identifier string
	name       string
	plan       int32
	joinedAt   time.Time
}

// NewSpaceBuilder は SpaceBuilder を生成します
func NewSpaceBuilder(t *testing.T, tx *sql.Tx) *SpaceBuilder {
	t.Helper()
	now := time.Now()
	return &SpaceBuilder{
		t:          t,
		tx:         tx,
		identifier: "test-space",
		name:       "Test Space",
		plan:       1, // small
		joinedAt:   now,
	}
}

// WithIdentifier は識別子を設定します
func (b *SpaceBuilder) WithIdentifier(identifier string) *SpaceBuilder {
	b.identifier = identifier
	return b
}

// WithName は名前を設定します
func (b *SpaceBuilder) WithName(name string) *SpaceBuilder {
	b.name = name
	return b
}

// WithPlan はプランを設定します
func (b *SpaceBuilder) WithPlan(plan int32) *SpaceBuilder {
	b.plan = plan
	return b
}

// Build はスペースを作成し、IDを返します
func (b *SpaceBuilder) Build() model.SpaceID {
	b.t.Helper()

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO spaces (identifier, name, plan, joined_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		b.identifier, b.name, b.plan, b.joinedAt, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("スペース作成に失敗: %v", err)
	}

	return model.SpaceID(id)
}
