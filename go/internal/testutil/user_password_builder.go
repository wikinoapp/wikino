package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// UserPasswordBuilder はユーザーパスワードテストデータのビルダー
type UserPasswordBuilder struct {
	t  *testing.T
	tx *sql.Tx

	userID         string
	passwordDigest string
}

// NewUserPasswordBuilder は UserPasswordBuilder を生成します
func NewUserPasswordBuilder(t *testing.T, tx *sql.Tx) *UserPasswordBuilder {
	t.Helper()
	return &UserPasswordBuilder{
		t:              t,
		tx:             tx,
		passwordDigest: "$2a$10$defaulthashedpassword12345",
	}
}

// WithUserID はユーザーIDを設定します
func (b *UserPasswordBuilder) WithUserID(userID string) *UserPasswordBuilder {
	b.userID = userID
	return b
}

// WithPasswordDigest はパスワードダイジェストを設定します
func (b *UserPasswordBuilder) WithPasswordDigest(passwordDigest string) *UserPasswordBuilder {
	b.passwordDigest = passwordDigest
	return b
}

// Build はユーザーパスワードを作成し、IDを返します
func (b *UserPasswordBuilder) Build() string {
	b.t.Helper()

	if b.userID == "" {
		b.t.Fatal("userID is required. Use WithUserID() to set it.")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO user_passwords (user_id, password_digest, created_at, updated_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		b.userID, b.passwordDigest, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("ユーザーパスワード作成に失敗: %v", err)
	}

	return id
}

// UserPasswordBuilderDB はDBを直接使用するユーザーパスワードテストデータのビルダー
// トランザクション管理を自前で行うUsecaseのテストに使用します
type UserPasswordBuilderDB struct {
	t  *testing.T
	db *sql.DB

	userID         string
	passwordDigest string
}

// NewUserPasswordBuilderDB は UserPasswordBuilderDB を生成します
func NewUserPasswordBuilderDB(t *testing.T, db *sql.DB) *UserPasswordBuilderDB {
	t.Helper()
	return &UserPasswordBuilderDB{
		t:              t,
		db:             db,
		passwordDigest: "$2a$10$defaulthashedpassword12345",
	}
}

// WithUserID はユーザーIDを設定します
func (b *UserPasswordBuilderDB) WithUserID(userID string) *UserPasswordBuilderDB {
	b.userID = userID
	return b
}

// WithPasswordDigest はパスワードダイジェストを設定します
func (b *UserPasswordBuilderDB) WithPasswordDigest(passwordDigest string) *UserPasswordBuilderDB {
	b.passwordDigest = passwordDigest
	return b
}

// Build はユーザーパスワードを作成し、IDを返します
func (b *UserPasswordBuilderDB) Build() string {
	b.t.Helper()

	if b.userID == "" {
		b.t.Fatal("userID is required. Use WithUserID() to set it.")
	}

	now := time.Now()
	var id string
	err := b.db.QueryRowContext(
		context.Background(),
		`INSERT INTO user_passwords (user_id, password_digest, created_at, updated_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		b.userID, b.passwordDigest, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("ユーザーパスワード作成に失敗: %v", err)
	}

	return id
}
