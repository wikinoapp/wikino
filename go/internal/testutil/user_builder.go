package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// UserBuilder はユーザーテストデータのビルダー
type UserBuilder struct {
	t  *testing.T
	tx *sql.Tx

	email       string
	atname      string
	name        string
	description string
	locale      int32
	timeZone    string
	joinedAt    time.Time
}

// NewUserBuilder は UserBuilder を生成します
func NewUserBuilder(t *testing.T, tx *sql.Tx) *UserBuilder {
	t.Helper()
	now := time.Now()
	return &UserBuilder{
		t:           t,
		tx:          tx,
		email:       "test@example.com",
		atname:      "testuser",
		name:        "Test User",
		description: "Test description",
		locale:      0, // ja
		timeZone:    "Asia/Tokyo",
		joinedAt:    now,
	}
}

// WithEmail はメールアドレスを設定します
func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.email = email
	return b
}

// WithAtname はアットネームを設定します
func (b *UserBuilder) WithAtname(atname string) *UserBuilder {
	b.atname = atname
	return b
}

// WithName は名前を設定します
func (b *UserBuilder) WithName(name string) *UserBuilder {
	b.name = name
	return b
}

// Build はユーザーを作成し、IDを返します
func (b *UserBuilder) Build() string {
	b.t.Helper()

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO users (email, atname, name, description, locale, time_zone, joined_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		b.email, b.atname, b.name, b.description, b.locale, b.timeZone, b.joinedAt, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("ユーザー作成に失敗: %v", err)
	}

	return id
}

// BuildWithPassword はユーザーとパスワードを作成し、ユーザーIDを返します
func (b *UserBuilder) BuildWithPassword(passwordDigest string) string {
	b.t.Helper()

	userID := b.Build()

	now := time.Now()
	_, err := b.tx.ExecContext(
		context.Background(),
		`INSERT INTO user_passwords (user_id, password_digest, created_at, updated_at)
		 VALUES ($1, $2, $3, $4)`,
		userID, passwordDigest, now, now,
	)
	if err != nil {
		b.t.Fatalf("ユーザーパスワード作成に失敗: %v", err)
	}

	return userID
}

// BuildWithTwoFactorAuth はユーザーと二要素認証設定を作成し、ユーザーIDを返します
func (b *UserBuilder) BuildWithTwoFactorAuth(secret string, enabled bool) string {
	b.t.Helper()

	userID := b.Build()

	now := time.Now()
	var enabledAt sql.NullTime
	if enabled {
		enabledAt = sql.NullTime{Time: now, Valid: true}
	}

	_, err := b.tx.ExecContext(
		context.Background(),
		`INSERT INTO user_two_factor_auths (user_id, secret, enabled, enabled_at, recovery_codes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, secret, enabled, enabledAt, "{}", now, now,
	)
	if err != nil {
		b.t.Fatalf("二要素認証設定作成に失敗: %v", err)
	}

	return userID
}

// QueriesWithTx はトランザクションを使用したQueriesを返します
func QueriesWithTx(tx *sql.Tx) *query.Queries {
	return query.New(tx)
}
