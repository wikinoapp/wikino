package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SessionBuilderはセッションテストデータのビルダー
type SessionBuilder struct {
	t  *testing.T
	tx *sql.Tx

	userID     model.UserID
	token      string
	ipAddress  string
	userAgent  string
	signedInAt time.Time
}

// NewSessionBuilderはSessionBuilderを生成します
func NewSessionBuilder(t *testing.T, tx *sql.Tx) *SessionBuilder {
	t.Helper()
	now := time.Now()
	return &SessionBuilder{
		t:          t,
		tx:         tx,
		token:      "test-session-token",
		ipAddress:  "127.0.0.1",
		userAgent:  "Mozilla/5.0 (Test)",
		signedInAt: now,
	}
}

// WithUserIDはユーザーIDを設定します
func (b *SessionBuilder) WithUserID(userID model.UserID) *SessionBuilder {
	b.userID = userID
	return b
}

// WithTokenはトークンを設定します
func (b *SessionBuilder) WithToken(token string) *SessionBuilder {
	b.token = token
	return b
}

// WithIPAddressはIPアドレスを設定します
func (b *SessionBuilder) WithIPAddress(ipAddress string) *SessionBuilder {
	b.ipAddress = ipAddress
	return b
}

// WithUserAgentはUser-Agentを設定します
func (b *SessionBuilder) WithUserAgent(userAgent string) *SessionBuilder {
	b.userAgent = userAgent
	return b
}

// Buildはセッションを作成し、IDを返します
func (b *SessionBuilder) Build() string {
	b.t.Helper()

	if b.userID == "" {
		b.t.Fatalf("セッション作成にはユーザーIDが必要です")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO user_sessions (user_id, token, ip_address, user_agent, signed_in_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		string(b.userID), b.token, b.ipAddress, b.userAgent, b.signedInAt, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("セッション作成に失敗: %v", err)
	}

	return id
}

// BuildAndGetTokenはセッションを作成し、トークンを返します
func (b *SessionBuilder) BuildAndGetToken() string {
	b.t.Helper()
	b.Build()
	return b.token
}
