package model

import (
	"time"
)

// OAuthRefreshTokenLifetimeはリフレッシュトークンの有効期間。ローテーションのたびに新しい
// トークンがこの期間を得るため、使い続けるクライアントは再認可を求められない
const OAuthRefreshTokenLifetime = 90 * 24 * time.Hour

// OAuthRefreshTokenはOAuthのリフレッシュトークンのドメインモデル。トークンは許可に属し、
// 使うたびにローテーションする。PreviousRefreshTokenIDはローテーション前のトークンを指す。
//
// トークンの値そのものは保持せず、TokenDigestで照合する。Scopesは更新で発行する
// アクセストークンのスコープである。
type OAuthRefreshToken struct {
	ID                     OAuthRefreshTokenID
	OAuthGrantID           OAuthGrantID
	SpaceID                SpaceID
	TokenDigest            string
	Scopes                 []Scope
	PreviousRefreshTokenID *OAuthRefreshTokenID
	ExpiresAt              time.Time
	UsedAt                 *time.Time
	RevokedAt              *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// IsExpiredはトークンの有効期限が切れているかを返す
func (t *OAuthRefreshToken) IsExpired(now time.Time) bool {
	return !now.Before(t.ExpiresAt)
}

// IsUsedはトークンがローテーションで使われたかを返す。使用済みのトークンが再び
// 提示されたら、漏洩したものとして許可に属するトークンをすべて失効させる
func (t *OAuthRefreshToken) IsUsed() bool {
	return t.UsedAt != nil
}

// IsRevokedはトークンが失効しているかを返す
func (t *OAuthRefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsActiveはトークンが未使用で、失効しておらず有効期限内であるかを返す。許可の失効や
// メンバーシップ、メンバー権限による有効性はここでは判定しない
func (t *OAuthRefreshToken) IsActive(now time.Time) bool {
	return !t.IsUsed() && !t.IsRevoked() && !t.IsExpired(now)
}
