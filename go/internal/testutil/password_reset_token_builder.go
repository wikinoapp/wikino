package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// PasswordResetTokenBuilder はパスワードリセットトークンテストデータのビルダー
type PasswordResetTokenBuilder struct {
	t  *testing.T
	tx *sql.Tx

	userID      model.UserID
	tokenDigest string
	expiresAt   time.Time
	usedAt      *time.Time
}

// NewPasswordResetTokenBuilder は PasswordResetTokenBuilder を生成します
func NewPasswordResetTokenBuilder(t *testing.T, tx *sql.Tx) *PasswordResetTokenBuilder {
	t.Helper()
	return &PasswordResetTokenBuilder{
		t:           t,
		tx:          tx,
		tokenDigest: "test_token_digest",
		expiresAt:   time.Now().Add(model.PasswordResetTokenExpirationDuration),
	}
}

// WithUserID はユーザーIDを設定します
func (b *PasswordResetTokenBuilder) WithUserID(userID model.UserID) *PasswordResetTokenBuilder {
	b.userID = userID
	return b
}

// WithTokenDigest はトークンダイジェストを設定します
func (b *PasswordResetTokenBuilder) WithTokenDigest(tokenDigest string) *PasswordResetTokenBuilder {
	b.tokenDigest = tokenDigest
	return b
}

// WithExpiresAt は有効期限を設定します
func (b *PasswordResetTokenBuilder) WithExpiresAt(expiresAt time.Time) *PasswordResetTokenBuilder {
	b.expiresAt = expiresAt
	return b
}

// WithUsedAt は使用日時を設定します
func (b *PasswordResetTokenBuilder) WithUsedAt(usedAt time.Time) *PasswordResetTokenBuilder {
	b.usedAt = &usedAt
	return b
}

// Build はパスワードリセットトークンを作成し、IDを返します
func (b *PasswordResetTokenBuilder) Build() string {
	b.t.Helper()

	if b.userID == "" {
		b.t.Fatal("userID is required. Use WithUserID() to set it.")
	}

	now := time.Now()
	var id string
	var err error

	if b.usedAt != nil {
		err = b.tx.QueryRowContext(
			context.Background(),
			`INSERT INTO password_reset_tokens (user_id, token_digest, expires_at, used_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id`,
			string(b.userID), b.tokenDigest, b.expiresAt, b.usedAt, now, now,
		).Scan(&id)
	} else {
		err = b.tx.QueryRowContext(
			context.Background(),
			`INSERT INTO password_reset_tokens (user_id, token_digest, expires_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id`,
			string(b.userID), b.tokenDigest, b.expiresAt, now, now,
		).Scan(&id)
	}
	if err != nil {
		b.t.Fatalf("パスワードリセットトークン作成に失敗: %v", err)
	}

	return id
}

// BuildUsed は使用済みのパスワードリセットトークンを作成し、IDを返します
func (b *PasswordResetTokenBuilder) BuildUsed() string {
	b.t.Helper()

	if b.userID == "" {
		b.t.Fatal("userID is required. Use WithUserID() to set it.")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO password_reset_tokens (user_id, token_digest, expires_at, used_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		string(b.userID), b.tokenDigest, b.expiresAt, now, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("パスワードリセットトークン作成に失敗: %v", err)
	}

	return id
}

// PasswordResetTokenBuilderDB はDBを直接使用するパスワードリセットトークンテストデータのビルダー
// トランザクション管理を自前で行うUsecaseのテストに使用します
type PasswordResetTokenBuilderDB struct {
	t  *testing.T
	db *sql.DB

	userID      model.UserID
	tokenDigest string
	expiresAt   time.Time
	usedAt      *time.Time
}

// NewPasswordResetTokenBuilderDB は PasswordResetTokenBuilderDB を生成します
func NewPasswordResetTokenBuilderDB(t *testing.T, db *sql.DB) *PasswordResetTokenBuilderDB {
	t.Helper()
	return &PasswordResetTokenBuilderDB{
		t:           t,
		db:          db,
		tokenDigest: "test_token_digest",
		expiresAt:   time.Now().Add(model.PasswordResetTokenExpirationDuration),
	}
}

// WithUserID はユーザーIDを設定します
func (b *PasswordResetTokenBuilderDB) WithUserID(userID model.UserID) *PasswordResetTokenBuilderDB {
	b.userID = userID
	return b
}

// WithTokenDigest はトークンダイジェストを設定します
func (b *PasswordResetTokenBuilderDB) WithTokenDigest(tokenDigest string) *PasswordResetTokenBuilderDB {
	b.tokenDigest = tokenDigest
	return b
}

// WithExpiresAt は有効期限を設定します
func (b *PasswordResetTokenBuilderDB) WithExpiresAt(expiresAt time.Time) *PasswordResetTokenBuilderDB {
	b.expiresAt = expiresAt
	return b
}

// WithUsedAt は使用日時を設定します
func (b *PasswordResetTokenBuilderDB) WithUsedAt(usedAt time.Time) *PasswordResetTokenBuilderDB {
	b.usedAt = &usedAt
	return b
}

// Build はパスワードリセットトークンを作成し、IDを返します
func (b *PasswordResetTokenBuilderDB) Build() string {
	b.t.Helper()

	if b.userID == "" {
		b.t.Fatal("userID is required. Use WithUserID() to set it.")
	}

	now := time.Now()
	var id string
	var err error

	if b.usedAt != nil {
		err = b.db.QueryRowContext(
			context.Background(),
			`INSERT INTO password_reset_tokens (user_id, token_digest, expires_at, used_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id`,
			string(b.userID), b.tokenDigest, b.expiresAt, b.usedAt, now, now,
		).Scan(&id)
	} else {
		err = b.db.QueryRowContext(
			context.Background(),
			`INSERT INTO password_reset_tokens (user_id, token_digest, expires_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id`,
			string(b.userID), b.tokenDigest, b.expiresAt, now, now,
		).Scan(&id)
	}
	if err != nil {
		b.t.Fatalf("パスワードリセットトークン作成に失敗: %v", err)
	}

	return id
}

// BuildUsed は使用済みのパスワードリセットトークンを作成し、IDを返します
func (b *PasswordResetTokenBuilderDB) BuildUsed() string {
	b.t.Helper()

	if b.userID == "" {
		b.t.Fatal("userID is required. Use WithUserID() to set it.")
	}

	now := time.Now()
	var id string
	err := b.db.QueryRowContext(
		context.Background(),
		`INSERT INTO password_reset_tokens (user_id, token_digest, expires_at, used_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		string(b.userID), b.tokenDigest, b.expiresAt, now, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("パスワードリセットトークン作成に失敗: %v", err)
	}

	return id
}
