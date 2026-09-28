package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// PersonalAccessTokenBuilderは個人アクセストークンテストデータのビルダー
type PersonalAccessTokenBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID        string
	spaceMemberID  string
	name           string
	tokenDigest    string
	tokenLastChars string
	scopes         []string
	expiresAt      time.Time
	lastUsedAt     *time.Time
	revokedAt      *time.Time
}

// NewPersonalAccessTokenBuilderはPersonalAccessTokenBuilderを生成します
func NewPersonalAccessTokenBuilder(t *testing.T, tx *sql.Tx) *PersonalAccessTokenBuilder {
	t.Helper()
	return &PersonalAccessTokenBuilder{
		t:              t,
		tx:             tx,
		name:           "テスト用トークン",
		tokenDigest:    "test_personal_access_token_digest",
		tokenLastChars: "abcd",
		scopes:         []string{string(model.ScopePageRead)},
		expiresAt:      time.Now().Add(30 * 24 * time.Hour),
	}
}

// WithSpaceIDはスペースIDを設定します
func (b *PersonalAccessTokenBuilder) WithSpaceID(spaceID model.SpaceID) *PersonalAccessTokenBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithSpaceMemberIDはトークンを持つスペースメンバーのIDを設定します
func (b *PersonalAccessTokenBuilder) WithSpaceMemberID(spaceMemberID model.SpaceMemberID) *PersonalAccessTokenBuilder {
	b.spaceMemberID = string(spaceMemberID)
	return b
}

// WithNameはトークンの名前を設定します
func (b *PersonalAccessTokenBuilder) WithName(name string) *PersonalAccessTokenBuilder {
	b.name = name
	return b
}

// WithTokenDigestはトークンダイジェストを設定します。token_digestは一意なので、1つの
// テストで複数のトークンを作る場合は、それぞれに異なる値を設定してください
func (b *PersonalAccessTokenBuilder) WithTokenDigest(tokenDigest string) *PersonalAccessTokenBuilder {
	b.tokenDigest = tokenDigest
	return b
}

// WithScopesはトークンのスコープを設定します
func (b *PersonalAccessTokenBuilder) WithScopes(scopes []model.Scope) *PersonalAccessTokenBuilder {
	b.scopes = model.ScopesToStrings(scopes)
	return b
}

// WithExpiresAtは有効期限を設定します
func (b *PersonalAccessTokenBuilder) WithExpiresAt(expiresAt time.Time) *PersonalAccessTokenBuilder {
	b.expiresAt = expiresAt
	return b
}

// WithLastUsedAtは最終使用日時を設定します
func (b *PersonalAccessTokenBuilder) WithLastUsedAt(lastUsedAt time.Time) *PersonalAccessTokenBuilder {
	b.lastUsedAt = &lastUsedAt
	return b
}

// WithRevokedAtは失効日時を設定します
func (b *PersonalAccessTokenBuilder) WithRevokedAt(revokedAt time.Time) *PersonalAccessTokenBuilder {
	b.revokedAt = &revokedAt
	return b
}

// Buildは個人アクセストークンを作成し、IDを返します
func (b *PersonalAccessTokenBuilder) Build() model.PersonalAccessTokenID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("PersonalAccessTokenBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.spaceMemberID == "" {
		b.t.Fatal("PersonalAccessTokenBuilder: spaceMemberIDが設定されていません。WithSpaceMemberID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO personal_access_tokens (space_id, space_member_id, name, token_digest, token_last_chars, scopes, expires_at, last_used_at, revoked_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		 RETURNING id`,
		b.spaceID, b.spaceMemberID, b.name, b.tokenDigest, b.tokenLastChars, pq.Array(b.scopes), b.expiresAt, b.lastUsedAt, b.revokedAt, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("個人アクセストークン作成に失敗: %v", err)
	}

	return model.PersonalAccessTokenID(id)
}
