package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// OAuthAccessTokenBuilderはOAuthのアクセストークンテストデータのビルダー
type OAuthAccessTokenBuilder struct {
	t  *testing.T
	tx *sql.Tx

	oauthGrantID string
	spaceID      string
	tokenDigest  string
	scopes       []string
	expiresAt    time.Time
	revokedAt    *time.Time
}

// NewOAuthAccessTokenBuilderはOAuthAccessTokenBuilderを生成します
func NewOAuthAccessTokenBuilder(t *testing.T, tx *sql.Tx) *OAuthAccessTokenBuilder {
	t.Helper()
	return &OAuthAccessTokenBuilder{
		t:           t,
		tx:          tx,
		tokenDigest: "test_oauth_access_token_digest",
		scopes:      []string{string(model.ScopePageRead)},
		expiresAt:   time.Now().Add(time.Hour),
	}
}

// WithOAuthGrantIDはトークンが属する許可のIDを設定します
func (b *OAuthAccessTokenBuilder) WithOAuthGrantID(oauthGrantID model.OAuthGrantID) *OAuthAccessTokenBuilder {
	b.oauthGrantID = string(oauthGrantID)
	return b
}

// WithSpaceIDはスペースIDを設定します。許可のスペースと同じ値にしてください
func (b *OAuthAccessTokenBuilder) WithSpaceID(spaceID model.SpaceID) *OAuthAccessTokenBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithTokenDigestはトークンダイジェストを設定します。token_digestは一意なので、1つの
// テストで複数のトークンを作る場合は、それぞれに異なる値を設定してください
func (b *OAuthAccessTokenBuilder) WithTokenDigest(tokenDigest string) *OAuthAccessTokenBuilder {
	b.tokenDigest = tokenDigest
	return b
}

// WithScopesはトークンのスコープを設定します
func (b *OAuthAccessTokenBuilder) WithScopes(scopes []model.Scope) *OAuthAccessTokenBuilder {
	b.scopes = model.ScopesToStrings(scopes)
	return b
}

// WithExpiresAtは有効期限を設定します
func (b *OAuthAccessTokenBuilder) WithExpiresAt(expiresAt time.Time) *OAuthAccessTokenBuilder {
	b.expiresAt = expiresAt
	return b
}

// WithRevokedAtは失効日時を設定します
func (b *OAuthAccessTokenBuilder) WithRevokedAt(revokedAt time.Time) *OAuthAccessTokenBuilder {
	b.revokedAt = &revokedAt
	return b
}

// Buildはアクセストークンを作成し、IDを返します
func (b *OAuthAccessTokenBuilder) Build() model.OAuthAccessTokenID {
	b.t.Helper()

	if b.oauthGrantID == "" {
		b.t.Fatal("OAuthAccessTokenBuilder: oauthGrantIDが設定されていません。WithOAuthGrantID()を呼んでください")
	}
	if b.spaceID == "" {
		b.t.Fatal("OAuthAccessTokenBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO oauth_access_tokens (oauth_grant_id, space_id, token_digest, scopes, expires_at, revoked_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		 RETURNING id`,
		b.oauthGrantID, b.spaceID, b.tokenDigest, pq.Array(b.scopes), b.expiresAt, b.revokedAt, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("アクセストークン作成に失敗: %v", err)
	}

	return model.OAuthAccessTokenID(id)
}
