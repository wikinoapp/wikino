package model

import (
	"time"
)

// OAuthRefreshTokenLifetimeはリフレッシュトークンの有効期間。ローテーションのたびに新しい
// トークンがこの期間を得るため、使い続けるクライアントは再認可を求められない
const OAuthRefreshTokenLifetime = 90 * 24 * time.Hour

// OAuthRefreshTokenReuseIntervalは、使用済みのリフレッシュトークンの再提示を、漏洩ではなく
// 更新の再送として扱う猶予。クライアントが更新の応答を通信の失敗で受け取れずに同じトークンを
// 再送しても、連携を失効させずに新しいトークンを発行する。長さはOktaの既定 (30秒) に揃えた
const OAuthRefreshTokenReuseInterval = 30 * time.Second

// OAuthRefreshTokenはOAuthのリフレッシュトークンのドメインモデル。トークンは許可に属し、
// 使うたびにローテーションする。PreviousRefreshTokenIDはローテーション前のトークンを指す。
//
// トークンの値そのものは保持せず、TokenDigestで照合する。Scopesは更新で発行する
// アクセストークンのスコープ、Audienceは更新で発行するアクセストークンと次のリフレッシュトークンの宛先である。
type OAuthRefreshToken struct {
	ID                     OAuthRefreshTokenID
	OAuthGrantID           OAuthGrantID
	SpaceID                SpaceID
	TokenDigest            string
	Scopes                 []Scope
	Audience               OAuthAudience
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

// IsUsedはトークンがローテーションで使われたかを返す。使用済みのトークンが猶予を過ぎて再び
// 提示されたら、漏洩したものとして許可に属するトークンをすべて失効させる
func (t *OAuthRefreshToken) IsUsed() bool {
	return t.UsedAt != nil
}

// IsInReuseIntervalは、トークンが使われてから猶予 (OAuthRefreshTokenReuseInterval) の間に
// あるかを返す。猶予の間の再提示は、後継のトークンがまだ使われていなければ更新の再送として扱う
func (t *OAuthRefreshToken) IsInReuseInterval(now time.Time) bool {
	return t.UsedAt != nil && now.Before(t.UsedAt.Add(OAuthRefreshTokenReuseInterval))
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
