package model

import (
	"time"
)

// OAuthGrantは、スペースメンバーがOAuthアプリに与えた許可のドメインモデル。許可は1つの
// スペースに束縛され、認可コード・アクセストークン・リフレッシュトークンはどれも許可に属する。
// 同じアプリ・メンバーに失効していない許可は1つに限る。
type OAuthGrant struct {
	ID                 OAuthGrantID
	OAuthApplicationID OAuthApplicationID
	SpaceID            SpaceID
	SpaceMemberID      SpaceMemberID
	Scopes             []Scope
	RevokedAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// IsRevokedは許可が失効しているかを返す
func (g *OAuthGrant) IsRevoked() bool {
	return g.RevokedAt != nil
}
