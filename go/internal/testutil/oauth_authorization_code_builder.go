package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// OAuthAuthorizationCodeBuilderはOAuthの認可コードテストデータのビルダー
type OAuthAuthorizationCodeBuilder struct {
	t  *testing.T
	tx *sql.Tx

	oauthGrantID  string
	spaceID       string
	codeDigest    string
	scopes        []string
	redirectURI   string
	codeChallenge string
	expiresAt     time.Time
	usedAt        *time.Time
}

// NewOAuthAuthorizationCodeBuilderはOAuthAuthorizationCodeBuilderを生成します
func NewOAuthAuthorizationCodeBuilder(t *testing.T, tx *sql.Tx) *OAuthAuthorizationCodeBuilder {
	t.Helper()
	return &OAuthAuthorizationCodeBuilder{
		t:             t,
		tx:            tx,
		codeDigest:    "test_oauth_authorization_code_digest",
		scopes:        []string{string(model.ScopePageRead)},
		redirectURI:   "http://127.0.0.1/callback",
		codeChallenge: "test_code_challenge",
		expiresAt:     time.Now().Add(5 * time.Minute),
	}
}

// WithOAuthGrantIDはコードが属する許可のIDを設定します
func (b *OAuthAuthorizationCodeBuilder) WithOAuthGrantID(oauthGrantID model.OAuthGrantID) *OAuthAuthorizationCodeBuilder {
	b.oauthGrantID = string(oauthGrantID)
	return b
}

// WithSpaceIDはスペースIDを設定します。許可のスペースと同じ値にしてください
func (b *OAuthAuthorizationCodeBuilder) WithSpaceID(spaceID model.SpaceID) *OAuthAuthorizationCodeBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithCodeDigestはコードダイジェストを設定します。code_digestは一意なので、1つのテストで
// 複数のコードを作る場合は、それぞれに異なる値を設定してください
func (b *OAuthAuthorizationCodeBuilder) WithCodeDigest(codeDigest string) *OAuthAuthorizationCodeBuilder {
	b.codeDigest = codeDigest
	return b
}

// WithScopesはコードのスコープを設定します
func (b *OAuthAuthorizationCodeBuilder) WithScopes(scopes []model.Scope) *OAuthAuthorizationCodeBuilder {
	b.scopes = model.ScopesToStrings(scopes)
	return b
}

// WithRedirectURIは認可要求のリダイレクトURIを設定します
func (b *OAuthAuthorizationCodeBuilder) WithRedirectURI(redirectURI string) *OAuthAuthorizationCodeBuilder {
	b.redirectURI = redirectURI
	return b
}

// WithCodeChallengeはPKCEのcode_challenge (S256で計算した値) を設定します
func (b *OAuthAuthorizationCodeBuilder) WithCodeChallenge(codeChallenge string) *OAuthAuthorizationCodeBuilder {
	b.codeChallenge = codeChallenge
	return b
}

// WithExpiresAtは有効期限を設定します
func (b *OAuthAuthorizationCodeBuilder) WithExpiresAt(expiresAt time.Time) *OAuthAuthorizationCodeBuilder {
	b.expiresAt = expiresAt
	return b
}

// WithUsedAtは使用日時を設定します
func (b *OAuthAuthorizationCodeBuilder) WithUsedAt(usedAt time.Time) *OAuthAuthorizationCodeBuilder {
	b.usedAt = &usedAt
	return b
}

// Buildは認可コードを作成し、IDを返します
func (b *OAuthAuthorizationCodeBuilder) Build() model.OAuthAuthorizationCodeID {
	b.t.Helper()

	if b.oauthGrantID == "" {
		b.t.Fatal("OAuthAuthorizationCodeBuilder: oauthGrantIDが設定されていません。WithOAuthGrantID()を呼んでください")
	}
	if b.spaceID == "" {
		b.t.Fatal("OAuthAuthorizationCodeBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO oauth_authorization_codes (oauth_grant_id, space_id, code_digest, scopes, redirect_uri, code_challenge, expires_at, used_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		 RETURNING id`,
		b.oauthGrantID, b.spaceID, b.codeDigest, pq.Array(b.scopes), b.redirectURI, b.codeChallenge, b.expiresAt, b.usedAt, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("認可コード作成に失敗: %v", err)
	}

	return model.OAuthAuthorizationCodeID(id)
}
