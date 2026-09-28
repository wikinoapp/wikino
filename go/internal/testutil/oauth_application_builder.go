package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// OAuthApplicationBuilderはOAuthアプリテストデータのビルダー
type OAuthApplicationBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID              *string
	createdSpaceMemberID *string
	officialClient       bool
	name                 string
	clientID             string
	clientSecretDigest   *string
	clientType           model.OAuthClientType
	redirectURIs         []string
	discardedAt          *time.Time
}

// NewOAuthApplicationBuilderはOAuthApplicationBuilderを生成します。既定ではpublicクライアントを作ります。
// スペースのアプリはWithSpaceID()、公式クライアントはAsOfficialClient()で作ります
func NewOAuthApplicationBuilder(t *testing.T, tx *sql.Tx) *OAuthApplicationBuilder {
	t.Helper()
	return &OAuthApplicationBuilder{
		t:            t,
		tx:           tx,
		name:         "テスト用アプリ",
		clientID:     "test_oauth_client_id",
		clientType:   model.OAuthClientTypePublic,
		redirectURIs: []string{"http://127.0.0.1/callback"},
	}
}

// WithSpaceIDはアプリを持つスペースのIDを設定します
func (b *OAuthApplicationBuilder) WithSpaceID(spaceID model.SpaceID) *OAuthApplicationBuilder {
	id := string(spaceID)
	b.spaceID = &id
	return b
}

// WithCreatedSpaceMemberIDはアプリを作成したスペースメンバーのIDを設定します
func (b *OAuthApplicationBuilder) WithCreatedSpaceMemberID(spaceMemberID model.SpaceMemberID) *OAuthApplicationBuilder {
	id := string(spaceMemberID)
	b.createdSpaceMemberID = &id
	return b
}

// AsOfficialClientは、スペースに属さない全スペース共通の公式クライアントにします
func (b *OAuthApplicationBuilder) AsOfficialClient() *OAuthApplicationBuilder {
	b.officialClient = true
	return b
}

// WithNameはアプリの名前を設定します
func (b *OAuthApplicationBuilder) WithName(name string) *OAuthApplicationBuilder {
	b.name = name
	return b
}

// WithClientIDはクライアントIDを設定します。client_idは一意なので、1つのテストで複数の
// アプリを作る場合は、それぞれに異なる値を設定してください
func (b *OAuthApplicationBuilder) WithClientID(clientID string) *OAuthApplicationBuilder {
	b.clientID = clientID
	return b
}

// WithConfidentialClientSecretDigestは、シークレットのダイジェストを持つconfidentialクライアントにします
func (b *OAuthApplicationBuilder) WithConfidentialClientSecretDigest(clientSecretDigest string) *OAuthApplicationBuilder {
	b.clientType = model.OAuthClientTypeConfidential
	b.clientSecretDigest = &clientSecretDigest
	return b
}

// WithRedirectURIsはリダイレクトURIを設定します
func (b *OAuthApplicationBuilder) WithRedirectURIs(redirectURIs []string) *OAuthApplicationBuilder {
	b.redirectURIs = redirectURIs
	return b
}

// WithDiscardedAtは削除日時を設定します
func (b *OAuthApplicationBuilder) WithDiscardedAt(discardedAt time.Time) *OAuthApplicationBuilder {
	b.discardedAt = &discardedAt
	return b
}

// BuildはOAuthアプリを作成し、IDを返します
func (b *OAuthApplicationBuilder) Build() model.OAuthApplicationID {
	b.t.Helper()

	if b.spaceID == nil && !b.officialClient {
		b.t.Fatal("OAuthApplicationBuilder: spaceIDが設定されていません。WithSpaceID()かAsOfficialClient()を呼んでください")
	}
	if b.spaceID != nil && b.officialClient {
		b.t.Fatal("OAuthApplicationBuilder: 公式クライアントにはspaceIDを設定できません")
	}

	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO oauth_applications (space_id, created_space_member_id, name, client_id, client_secret_digest, client_type, redirect_uris, discarded_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		 RETURNING id`,
		b.spaceID, b.createdSpaceMemberID, b.name, b.clientID, b.clientSecretDigest, int32(b.clientType), pq.Array(b.redirectURIs), b.discardedAt, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("OAuthアプリ作成に失敗: %v", err)
	}

	return model.OAuthApplicationID(id)
}
