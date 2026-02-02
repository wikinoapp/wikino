package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// EmailConfirmationBuilder はメール確認テストデータのビルダー
type EmailConfirmationBuilder struct {
	t  *testing.T
	tx *sql.Tx

	email       string
	event       model.EmailConfirmationEvent
	code        string
	startedAt   time.Time
	succeededAt *time.Time
}

// NewEmailConfirmationBuilder は EmailConfirmationBuilder を生成します
func NewEmailConfirmationBuilder(t *testing.T, tx *sql.Tx) *EmailConfirmationBuilder {
	t.Helper()
	return &EmailConfirmationBuilder{
		t:         t,
		tx:        tx,
		email:     "test@example.com",
		event:     model.EmailConfirmationEventSignUp,
		code:      "ABC123",
		startedAt: time.Now(),
	}
}

// WithEmail はメールアドレスを設定します
func (b *EmailConfirmationBuilder) WithEmail(email string) *EmailConfirmationBuilder {
	b.email = email
	return b
}

// WithEvent はイベント種別を設定します
func (b *EmailConfirmationBuilder) WithEvent(event model.EmailConfirmationEvent) *EmailConfirmationBuilder {
	b.event = event
	return b
}

// WithCode は確認コードを設定します
func (b *EmailConfirmationBuilder) WithCode(code string) *EmailConfirmationBuilder {
	b.code = code
	return b
}

// WithStartedAt は開始日時を設定します
func (b *EmailConfirmationBuilder) WithStartedAt(startedAt time.Time) *EmailConfirmationBuilder {
	b.startedAt = startedAt
	return b
}

// WithSucceededAt は確認完了日時を設定します
func (b *EmailConfirmationBuilder) WithSucceededAt(succeededAt time.Time) *EmailConfirmationBuilder {
	b.succeededAt = &succeededAt
	return b
}

// Build はメール確認情報を作成し、IDを返します
func (b *EmailConfirmationBuilder) Build() string {
	b.t.Helper()

	now := time.Now()
	var id string
	var err error

	if b.succeededAt != nil {
		err = b.tx.QueryRowContext(
			context.Background(),
			`INSERT INTO email_confirmations (email, event, code, started_at, succeeded_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id`,
			b.email, int32(b.event), b.code, b.startedAt, b.succeededAt, now, now,
		).Scan(&id)
	} else {
		err = b.tx.QueryRowContext(
			context.Background(),
			`INSERT INTO email_confirmations (email, event, code, started_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id`,
			b.email, int32(b.event), b.code, b.startedAt, now, now,
		).Scan(&id)
	}
	if err != nil {
		b.t.Fatalf("メール確認情報作成に失敗: %v", err)
	}

	return id
}

// BuildSucceeded は確認完了状態のメール確認情報を作成し、IDを返します
func (b *EmailConfirmationBuilder) BuildSucceeded() string {
	b.t.Helper()

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO email_confirmations (email, event, code, started_at, succeeded_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.email, int32(b.event), b.code, b.startedAt, now, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("メール確認情報作成に失敗: %v", err)
	}

	return id
}

// EmailConfirmationBuilderDB はDBを直接使用するメール確認テストデータのビルダー
// トランザクション管理を自前で行うUsecaseのテストに使用します
type EmailConfirmationBuilderDB struct {
	t  *testing.T
	db *sql.DB

	email       string
	event       model.EmailConfirmationEvent
	code        string
	startedAt   time.Time
	succeededAt *time.Time
}

// NewEmailConfirmationBuilderDB は EmailConfirmationBuilderDB を生成します
func NewEmailConfirmationBuilderDB(t *testing.T, db *sql.DB) *EmailConfirmationBuilderDB {
	t.Helper()
	return &EmailConfirmationBuilderDB{
		t:         t,
		db:        db,
		email:     "test@example.com",
		event:     model.EmailConfirmationEventSignUp,
		code:      "ABC123",
		startedAt: time.Now(),
	}
}

// WithEmail はメールアドレスを設定します
func (b *EmailConfirmationBuilderDB) WithEmail(email string) *EmailConfirmationBuilderDB {
	b.email = email
	return b
}

// WithEvent はイベント種別を設定します
func (b *EmailConfirmationBuilderDB) WithEvent(event model.EmailConfirmationEvent) *EmailConfirmationBuilderDB {
	b.event = event
	return b
}

// WithCode は確認コードを設定します
func (b *EmailConfirmationBuilderDB) WithCode(code string) *EmailConfirmationBuilderDB {
	b.code = code
	return b
}

// WithStartedAt は開始日時を設定します
func (b *EmailConfirmationBuilderDB) WithStartedAt(startedAt time.Time) *EmailConfirmationBuilderDB {
	b.startedAt = startedAt
	return b
}

// WithSucceededAt は確認完了日時を設定します
func (b *EmailConfirmationBuilderDB) WithSucceededAt(succeededAt time.Time) *EmailConfirmationBuilderDB {
	b.succeededAt = &succeededAt
	return b
}

// Build はメール確認情報を作成し、IDを返します
func (b *EmailConfirmationBuilderDB) Build() string {
	b.t.Helper()

	now := time.Now()
	var id string
	var err error

	if b.succeededAt != nil {
		err = b.db.QueryRowContext(
			context.Background(),
			`INSERT INTO email_confirmations (email, event, code, started_at, succeeded_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id`,
			b.email, int32(b.event), b.code, b.startedAt, b.succeededAt, now, now,
		).Scan(&id)
	} else {
		err = b.db.QueryRowContext(
			context.Background(),
			`INSERT INTO email_confirmations (email, event, code, started_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id`,
			b.email, int32(b.event), b.code, b.startedAt, now, now,
		).Scan(&id)
	}
	if err != nil {
		b.t.Fatalf("メール確認情報作成に失敗: %v", err)
	}

	return id
}

// BuildSucceeded は確認完了状態のメール確認情報を作成し、IDを返します
func (b *EmailConfirmationBuilderDB) BuildSucceeded() string {
	b.t.Helper()

	now := time.Now()
	var id string
	err := b.db.QueryRowContext(
		context.Background(),
		`INSERT INTO email_confirmations (email, event, code, started_at, succeeded_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		b.email, int32(b.event), b.code, b.startedAt, now, now, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("メール確認情報作成に失敗: %v", err)
	}

	return id
}
