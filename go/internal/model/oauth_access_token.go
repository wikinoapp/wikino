package model

import (
	"time"
)

// OAuthAccessTokenLifetimeはアクセストークンの有効期間。リクエストのたびにDBで照合するため
// 失効は即時に効くが、漏れたトークンが使える時間の上限としても短くする
const OAuthAccessTokenLifetime = time.Hour

// OAuthAccessTokenはOAuthのアクセストークンのドメインモデル。トークンは許可に属し、
// 持ち主のスペースメンバーは許可から決まる。
//
// トークンの値そのものは保持せず、TokenDigestで照合する。
type OAuthAccessToken struct {
	ID           OAuthAccessTokenID
	OAuthGrantID OAuthGrantID
	SpaceID      SpaceID
	TokenDigest  string
	Scopes       []Scope
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsExpiredはトークンの有効期限が切れているかを返す
func (t *OAuthAccessToken) IsExpired(now time.Time) bool {
	return !now.Before(t.ExpiresAt)
}

// IsRevokedはトークンが失効しているかを返す
func (t *OAuthAccessToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsActiveはトークンが失効しておらず有効期限内であるかを返す。許可の失効やメンバーシップ、
// メンバー権限による有効性はここでは判定しない
func (t *OAuthAccessToken) IsActive(now time.Time) bool {
	return !t.IsRevoked() && !t.IsExpired(now)
}
