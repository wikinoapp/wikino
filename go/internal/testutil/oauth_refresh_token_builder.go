package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// OAuthRefreshTokenBuilderはOAuthのリフレッシュトークンテストデータのビルダー
type OAuthRefreshTokenBuilder struct {
	t  *testing.T
	tx *sql.Tx

	oauthGrantID string
	spaceID      string
	tokenDigest  string
	scopes       []string
	expiresAt    time.Time
	usedAt       *time.Time
	revokedAt    *time.Time
}

// NewOAuthRefreshTokenBuilderはOAuthRefreshTokenBuilderを生成します
func NewOAuthRefreshTokenBuilder(t *testing.T, tx *sql.Tx) *OAuthRefreshTokenBuilder {
	t.Helper()
	return &OAuthRefreshTokenBuilder{
		t:           t,
		tx:          tx,
		tokenDigest: "test_oauth_refresh_token_digest",
		scopes:      []string{string(model.ScopePageRead)},
		expiresAt:   time.Now().Add(90 * 24 * time.Hour),
	}
}

// WithOAuthGrantIDはトークンが属する許可のIDを設定します
func (b *OAuthRefreshTokenBuilder) WithOAuthGrantID(oauthGrantID model.OAuthGrantID) *OAuthRefreshTokenBuilder {
	b.oauthGrantID = string(oauthGrantID)
	return b
}

// WithSpaceIDはスペースIDを設定します。許可のスペースと同じ値にしてください
func (b *OAuthRefreshTokenBuilder) WithSpaceID(spaceID model.SpaceID) *OAuthRefreshTokenBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithTokenDigestはトークンダイジェストを設定します。token_digestは一意なので、1つの
// テストで複数のトークンを作る場合は、それぞれに異なる値を設定してください
func (b *OAuthRefreshTokenBuilder) WithTokenDigest(tokenDigest string) *OAuthRefreshTokenBuilder {
	b.tokenDigest = tokenDigest
	return b
}

// WithScopesはトークンのスコープを設定します
func (b *OAuthRefreshTokenBuilder) WithScopes(scopes []model.Scope) *OAuthRefreshTokenBuilder {
	b.scopes = model.ScopesToStrings(scopes)
	return b
}

// WithExpiresAtは有効期限を設定します
func (b *OAuthRefreshTokenBuilder) WithExpiresAt(expiresAt time.Time) *OAuthRefreshTokenBuilder {
	b.expiresAt = expiresAt
	return b
}

// WithUsedAtは使用日時を設定します
func (b *OAuthRefreshTokenBuilder) WithUsedAt(usedAt time.Time) *OAuthRefreshTokenBuilder {
	b.usedAt = &usedAt
	return b
}

// WithRevokedAtは失効日時を設定します
func (b *OAuthRefreshTokenBuilder) WithRevokedAt(revokedAt time.Time) *OAuthRefreshTokenBuilder {
	b.revokedAt = &revokedAt
	return b
}

// Buildはリフレッシュトークンを作成し、IDを返します
func (b *OAuthRefreshTokenBuilder) Build() model.OAuthRefreshTokenID {
	b.t.Helper()

	if b.oauthGrantID == "" {
		b.t.Fatal("OAuthRefreshTokenBuilder: oauthGrantIDが設定されていません。WithOAuthGrantID()を呼んでください")
	}
	if b.spaceID == "" {
		b.t.Fatal("OAuthRefreshTokenBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO oauth_refresh_tokens (oauth_grant_id, space_id, token_digest, scopes, expires_at, used_at, revoked_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		 RETURNING id`,
		b.oauthGrantID, b.spaceID, b.tokenDigest, pq.Array(b.scopes), b.expiresAt, b.usedAt, b.revokedAt, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("リフレッシュトークン作成に失敗: %v", err)
	}

	return model.OAuthRefreshTokenID(id)
}
