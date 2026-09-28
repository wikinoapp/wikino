package model

import (
	"time"
)

// OAuthAuthorizationCodeはOAuthの認可コードのドメインモデル。コードは1回だけ
// トークンと交換できる。
//
// コードの値そのものは保持せず、CodeDigestで照合する。PKCEはS256だけを受け付けるため、
// CodeChallengeはS256で計算した値である。RedirectURIは、トークン要求で一致を確かめる
// ための認可要求のリダイレクトURIである。
type OAuthAuthorizationCode struct {
	ID            OAuthAuthorizationCodeID
	OAuthGrantID  OAuthGrantID
	SpaceID       SpaceID
	CodeDigest    string
	Scopes        []Scope
	RedirectURI   string
	CodeChallenge string
	ExpiresAt     time.Time
	UsedAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// IsExpiredはコードの有効期限が切れているかを返す
func (c *OAuthAuthorizationCode) IsExpired(now time.Time) bool {
	return !now.Before(c.ExpiresAt)
}

// IsUsedはコードがトークンとの交換に使われたかを返す
func (c *OAuthAuthorizationCode) IsUsed() bool {
	return c.UsedAt != nil
}
