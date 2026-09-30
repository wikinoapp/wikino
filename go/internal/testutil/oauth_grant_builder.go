package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// OAuthGrantBuilderはOAuthアプリの許可テストデータのビルダー
type OAuthGrantBuilder struct {
	t  *testing.T
	tx *sql.Tx

	oauthApplicationID string
	spaceID            string
	spaceMemberID      string
	scopes             []string
	revokedAt          *time.Time
}

// NewOAuthGrantBuilderはOAuthGrantBuilderを生成します
func NewOAuthGrantBuilder(t *testing.T, tx *sql.Tx) *OAuthGrantBuilder {
	t.Helper()
	return &OAuthGrantBuilder{
		t:      t,
		tx:     tx,
		scopes: []string{string(model.ScopePageRead)},
	}
}

// WithOAuthApplicationIDは許可したOAuthアプリのIDを設定します
func (b *OAuthGrantBuilder) WithOAuthApplicationID(oauthApplicationID model.OAuthApplicationID) *OAuthGrantBuilder {
	b.oauthApplicationID = string(oauthApplicationID)
	return b
}

// WithSpaceIDはスペースIDを設定します
func (b *OAuthGrantBuilder) WithSpaceID(spaceID model.SpaceID) *OAuthGrantBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithSpaceMemberIDは許可したスペースメンバーのIDを設定します
func (b *OAuthGrantBuilder) WithSpaceMemberID(spaceMemberID model.SpaceMemberID) *OAuthGrantBuilder {
	b.spaceMemberID = string(spaceMemberID)
	return b
}

// WithScopesは許可のスコープを設定します
func (b *OAuthGrantBuilder) WithScopes(scopes []model.Scope) *OAuthGrantBuilder {
	b.scopes = model.ScopesToStrings(scopes)
	return b
}

// WithRevokedAtは失効日時を設定します
func (b *OAuthGrantBuilder) WithRevokedAt(revokedAt time.Time) *OAuthGrantBuilder {
	b.revokedAt = &revokedAt
	return b
}

// BuildはOAuthアプリの許可を作成し、IDを返します
func (b *OAuthGrantBuilder) Build() model.OAuthGrantID {
	b.t.Helper()

	if b.oauthApplicationID == "" {
		b.t.Fatal("OAuthGrantBuilder: oauthApplicationIDが設定されていません。WithOAuthApplicationID()を呼んでください")
	}
	if b.spaceID == "" {
		b.t.Fatal("OAuthGrantBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.spaceMemberID == "" {
		b.t.Fatal("OAuthGrantBuilder: spaceMemberIDが設定されていません。WithSpaceMemberID()を呼んでください")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO oauth_grants (oauth_application_id, space_id, space_member_id, scopes, revoked_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)
		 RETURNING id`,
		b.oauthApplicationID, b.spaceID, b.spaceMemberID, pq.Array(b.scopes), b.revokedAt, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("OAuthアプリの許可作成に失敗: %v", err)
	}

	return model.OAuthGrantID(id)
}
