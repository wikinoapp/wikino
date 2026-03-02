package testutil

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// FeatureFlagBuilder はフィーチャーフラグテストデータのビルダー
type FeatureFlagBuilder struct {
	t  *testing.T
	tx *sql.Tx

	userID model.UserID
	name   string
}

// NewFeatureFlagBuilder は FeatureFlagBuilder を生成します
func NewFeatureFlagBuilder(t *testing.T, tx *sql.Tx) *FeatureFlagBuilder {
	t.Helper()
	return &FeatureFlagBuilder{
		t:    t,
		tx:   tx,
		name: "test_flag",
	}
}

// WithUserID はユーザーIDを設定します
func (b *FeatureFlagBuilder) WithUserID(userID model.UserID) *FeatureFlagBuilder {
	b.userID = userID
	return b
}

// WithName はフラグ名を設定します
func (b *FeatureFlagBuilder) WithName(name string) *FeatureFlagBuilder {
	b.name = name
	return b
}

// Build はフィーチャーフラグを作成し、IDを返します
func (b *FeatureFlagBuilder) Build() model.FeatureFlagID {
	b.t.Helper()

	if b.userID == "" {
		b.t.Fatal("FeatureFlagBuilder: userIDが設定されていません。WithUserID()を呼んでください")
	}

	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO feature_flags (user_id, name)
		 VALUES ($1, $2)
		 RETURNING id`,
		string(b.userID), b.name,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("フィーチャーフラグ作成に失敗: %v", err)
	}

	return model.FeatureFlagID(id)
}
