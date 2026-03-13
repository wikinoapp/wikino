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

	deviceToken *string
	userID      *model.UserID
	name        string
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

// WithDeviceToken はデバイストークンを設定します
func (b *FeatureFlagBuilder) WithDeviceToken(deviceToken string) *FeatureFlagBuilder {
	b.deviceToken = &deviceToken
	return b
}

// WithUserID はユーザーIDを設定します
func (b *FeatureFlagBuilder) WithUserID(userID model.UserID) *FeatureFlagBuilder {
	b.userID = &userID
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

	if b.deviceToken == nil && b.userID == nil {
		b.t.Fatal("FeatureFlagBuilder: deviceTokenまたはuserIDのいずれかが必要です。WithDeviceToken()またはWithUserID()を呼んでください")
	}

	var deviceToken *string
	if b.deviceToken != nil {
		deviceToken = b.deviceToken
	}

	var userID *string
	if b.userID != nil {
		s := string(*b.userID)
		userID = &s
	}

	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO feature_flags (device_token, user_id, name)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		deviceToken, userID, b.name,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("フィーチャーフラグ作成に失敗: %v", err)
	}

	return model.FeatureFlagID(id)
}
