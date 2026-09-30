package model

import (
	"time"
)

// PersonalAccessTokenLastUsedAtUpdateIntervalは最終使用日時を更新する間隔。リクエストの
// たびに書き込まないよう、最終使用日時がこれより古いときだけ更新する
const PersonalAccessTokenLastUsedAtUpdateInterval = 1 * time.Minute

// PersonalAccessTokenExpirationDaysは個人アクセストークンの発行時に選べる有効期限の日数。
// 漏洩したトークンが使われ続けないよう、期限なしは選べない
var PersonalAccessTokenExpirationDays = []int{7, 30, 90, 365}

// PersonalAccessTokenは公開APIの個人アクセストークンのドメインモデル。トークンは1つの
// スペースに束縛され、そのスペースのメンバーが持つ。
//
// トークンの値そのものは保持せず、TokenDigestで照合する。TokenLastCharsは一覧で
// トークンを見分けるための、値の末尾の数文字である。
type PersonalAccessToken struct {
	ID             PersonalAccessTokenID
	SpaceID        SpaceID
	SpaceMemberID  SpaceMemberID
	Name           string
	TokenDigest    string
	TokenLastChars string
	Scopes         []Scope
	ExpiresAt      time.Time
	LastUsedAt     *time.Time
	RevokedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsExpiredはトークンの有効期限が切れているかを返す
func (t *PersonalAccessToken) IsExpired(now time.Time) bool {
	return !now.Before(t.ExpiresAt)
}

// IsRevokedはトークンが失効しているかを返す
func (t *PersonalAccessToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsActiveはトークンが失効しておらず有効期限内であるかを返す。メンバーシップやメンバー
// 権限による有効性はここでは判定しない
func (t *PersonalAccessToken) IsActive(now time.Time) bool {
	return !t.IsRevoked() && !t.IsExpired(now)
}
