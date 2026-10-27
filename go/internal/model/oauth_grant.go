package model

import (
	"time"
)

// OAuthGrantMaxPerApplicationMemberは、同じアプリ・メンバーの失効していない許可の上限。許可は認可ごとに
// 作るため、使われなくなった許可 (保存先を失った端末のものなど) がたまらないよう、超えるときは
// 使われていないものから押し出す
const OAuthGrantMaxPerApplicationMember = 10

// OAuthGrantEvictionGracePeriodは、まだ使われていない許可を押し出しの先頭に回すまでの猶予。
// 同意したばかりで認可コードをまだ交換していない許可 (ログインの途中) を先に押し出さないためにある
const OAuthGrantEvictionGracePeriod = 1 * time.Minute

// OAuthGrantLastUsedAtUpdateIntervalは、アクセストークンの照合で許可の最終利用日時を更新する間隔。
// リクエストのたびに書き込まないよう、前回の更新からこの間隔が過ぎたときだけ更新する
const OAuthGrantLastUsedAtUpdateInterval = 1 * time.Minute

// OAuthGrantは、スペースメンバーがOAuthアプリに与えた許可のドメインモデル。許可は1つの
// スペースに属し、認可コード・アクセストークン・リフレッシュトークンはどれも許可に属する。
// 許可は同意とデバイス認可の許可のたびに作るため、同じアプリ・メンバーに失効していない許可が
// 複数並びうる (端末ごとのログインなど)。
type OAuthGrant struct {
	ID                 OAuthGrantID
	OAuthApplicationID OAuthApplicationID
	SpaceID            SpaceID
	SpaceMemberID      SpaceMemberID
	Scopes             []Scope

	// DeviceNameは、連携の一覧で端末を見分けるためにクライアントが申告した端末名。nilは送られなかった
	// ことを表す。申告された値のため、同意画面には出さず、本人の連携の一覧にだけ出す
	DeviceName *string

	// LastUsedAtは最終利用日時 (トークンの発行かアクセストークンの照合)。nilはまだトークンを
	// 発行していないことを表す
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// IsRevokedは許可が失効しているかを返す
func (g *OAuthGrant) IsRevoked() bool {
	return g.RevokedAt != nil
}
