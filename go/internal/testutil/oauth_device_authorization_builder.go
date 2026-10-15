package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// OAuthDeviceAuthorizationBuilderはOAuthのデバイス認可の要求テストデータのビルダー
type OAuthDeviceAuthorizationBuilder struct {
	t  *testing.T
	tx *sql.Tx

	oauthApplicationID string
	spaceID            string
	oauthGrantID       *string
	deviceCodeDigest   string
	userCodeDigest     string
	scopes             []string
	status             model.OAuthDeviceAuthorizationStatus
	interval           time.Duration
	expiresAt          time.Time
	lastPolledAt       *time.Time
}

// NewOAuthDeviceAuthorizationBuilderはOAuthDeviceAuthorizationBuilderを生成します
func NewOAuthDeviceAuthorizationBuilder(t *testing.T, tx *sql.Tx) *OAuthDeviceAuthorizationBuilder {
	t.Helper()
	return &OAuthDeviceAuthorizationBuilder{
		t:                t,
		tx:               tx,
		deviceCodeDigest: "test_oauth_device_code_digest",
		userCodeDigest:   "test_oauth_user_code_digest",
		scopes:           []string{string(model.ScopePageRead)},
		status:           model.OAuthDeviceAuthorizationStatusPending,
		interval:         model.OAuthDeviceAuthorizationInterval,
		expiresAt:        time.Now().Add(model.OAuthDeviceAuthorizationLifetime),
	}
}

// WithOAuthApplicationIDは要求したOAuthアプリのIDを設定します
func (b *OAuthDeviceAuthorizationBuilder) WithOAuthApplicationID(oauthApplicationID model.OAuthApplicationID) *OAuthDeviceAuthorizationBuilder {
	b.oauthApplicationID = string(oauthApplicationID)
	return b
}

// WithSpaceIDは連携先のスペースIDを設定します
func (b *OAuthDeviceAuthorizationBuilder) WithSpaceID(spaceID model.SpaceID) *OAuthDeviceAuthorizationBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithOAuthGrantIDは許可のIDを設定します。許可済みか交換済みの状態と組み合わせてください
func (b *OAuthDeviceAuthorizationBuilder) WithOAuthGrantID(oauthGrantID model.OAuthGrantID) *OAuthDeviceAuthorizationBuilder {
	id := string(oauthGrantID)
	b.oauthGrantID = &id
	return b
}

// WithoutOAuthGrantIDは許可を外します。許可待ちか拒否の状態と組み合わせてください
func (b *OAuthDeviceAuthorizationBuilder) WithoutOAuthGrantID() *OAuthDeviceAuthorizationBuilder {
	b.oauthGrantID = nil
	return b
}

// WithDeviceCodeDigestはデバイスコードのダイジェストを設定します。device_code_digestは一意なので、
// 1つのテストで複数の要求を作る場合は、それぞれに異なる値を設定してください
func (b *OAuthDeviceAuthorizationBuilder) WithDeviceCodeDigest(deviceCodeDigest string) *OAuthDeviceAuthorizationBuilder {
	b.deviceCodeDigest = deviceCodeDigest
	return b
}

// WithUserCodeDigestはユーザーコードのダイジェストを設定します。user_code_digestは一意なので、
// 1つのテストで複数の要求を作る場合は、それぞれに異なる値を設定してください
func (b *OAuthDeviceAuthorizationBuilder) WithUserCodeDigest(userCodeDigest string) *OAuthDeviceAuthorizationBuilder {
	b.userCodeDigest = userCodeDigest
	return b
}

// WithScopesは要求のスコープを設定します
func (b *OAuthDeviceAuthorizationBuilder) WithScopes(scopes []model.Scope) *OAuthDeviceAuthorizationBuilder {
	b.scopes = model.ScopesToStrings(scopes)
	return b
}

// WithStatusは要求の状態を設定します
func (b *OAuthDeviceAuthorizationBuilder) WithStatus(status model.OAuthDeviceAuthorizationStatus) *OAuthDeviceAuthorizationBuilder {
	b.status = status
	return b
}

// WithIntervalはポーリングの間隔を設定します
func (b *OAuthDeviceAuthorizationBuilder) WithInterval(interval time.Duration) *OAuthDeviceAuthorizationBuilder {
	b.interval = interval
	return b
}

// WithExpiresAtは有効期限を設定します
func (b *OAuthDeviceAuthorizationBuilder) WithExpiresAt(expiresAt time.Time) *OAuthDeviceAuthorizationBuilder {
	b.expiresAt = expiresAt
	return b
}

// WithLastPolledAtは最後のポーリングの日時を設定します
func (b *OAuthDeviceAuthorizationBuilder) WithLastPolledAt(lastPolledAt time.Time) *OAuthDeviceAuthorizationBuilder {
	b.lastPolledAt = &lastPolledAt
	return b
}

// Buildはデバイス認可の要求を作成し、IDを返します
func (b *OAuthDeviceAuthorizationBuilder) Build() model.OAuthDeviceAuthorizationID {
	b.t.Helper()

	if b.oauthApplicationID == "" {
		b.t.Fatal("OAuthDeviceAuthorizationBuilder: oauthApplicationIDが設定されていません。WithOAuthApplicationID()を呼んでください")
	}
	if b.spaceID == "" {
		b.t.Fatal("OAuthDeviceAuthorizationBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}

	// #nosec G115 -- ポーリングの間隔は数秒から数十秒で、int32に収まる。
	intervalSeconds := int32(b.interval / time.Second)
	now := time.Now()
	var id string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO oauth_device_authorizations (oauth_application_id, space_id, oauth_grant_id, device_code_digest, user_code_digest, scopes, status, interval_seconds, expires_at, last_polled_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		 RETURNING id`,
		b.oauthApplicationID, b.spaceID, b.oauthGrantID, b.deviceCodeDigest, b.userCodeDigest, pq.Array(b.scopes),
		int32(b.status), intervalSeconds, b.expiresAt, b.lastPolledAt, now,
	).Scan(&id)
	if err != nil {
		b.t.Fatalf("デバイス認可の要求の作成に失敗: %v", err)
	}

	return model.OAuthDeviceAuthorizationID(id)
}
