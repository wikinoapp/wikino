package model

import (
	"time"
)

// OAuthDeviceAuthorizationLifetimeはデバイス認可の要求の有効期間。利用者が別の端末で
// ユーザーコードを入力して許可するまでの時間で、ユーザーコードを総当たりできる時間を限る
const OAuthDeviceAuthorizationLifetime = 10 * time.Minute

// OAuthDeviceAuthorizationIntervalは、クライアントがトークンエンドポイントをポーリングする
// 既定の間隔 (RFC 8628 §3.2)
const OAuthDeviceAuthorizationInterval = 5 * time.Second

// OAuthDeviceAuthorizationSlowDownIncrementは、間隔を空けずに届いたポーリングに `slow_down` を
// 返すときに、以降のポーリングの間隔を延ばす幅 (RFC 8628 §3.5)
const OAuthDeviceAuthorizationSlowDownIncrement = 5 * time.Second

// OAuthDeviceAuthorizationStatusはデバイス認可の要求の状態
type OAuthDeviceAuthorizationStatus int32

const (
	// OAuthDeviceAuthorizationStatusPendingは、利用者がまだ許可も拒否もしていないことを表す
	OAuthDeviceAuthorizationStatusPending OAuthDeviceAuthorizationStatus = 0
	// OAuthDeviceAuthorizationStatusApprovedは、利用者が許可し、トークンとの交換を待っていることを表す
	OAuthDeviceAuthorizationStatusApproved OAuthDeviceAuthorizationStatus = 1
	// OAuthDeviceAuthorizationStatusDeniedは、利用者が拒否したことを表す
	OAuthDeviceAuthorizationStatusDenied OAuthDeviceAuthorizationStatus = 2
	// OAuthDeviceAuthorizationStatusExchangedは、デバイスコードをトークンと交換したことを表す
	OAuthDeviceAuthorizationStatusExchanged OAuthDeviceAuthorizationStatus = 3
)

// OAuthDeviceAuthorizationは、デバイス認可 (RFC 8628) の要求のドメインモデル。
// 連携先のスペースは要求の時点で決まっている。
//
// デバイスコードとユーザーコードの値そのものは保持せず、ダイジェストで照合する。
// OAuthGrantIDは利用者が許可したときに作った許可で、許可待ちと拒否ではnilである。
// Scopesは要求されたスコープで、許可したときに利用者が選んだ範囲に置き換わる。
type OAuthDeviceAuthorization struct {
	ID                 OAuthDeviceAuthorizationID
	OAuthApplicationID OAuthApplicationID
	SpaceID            SpaceID
	OAuthGrantID       *OAuthGrantID
	DeviceCodeDigest   string
	UserCodeDigest     string
	Scopes             []Scope

	// DeviceNameは要求で受け取った、クライアントが申告した端末名。nilは送られなかったことを表す。
	// 利用者が許可したときに許可へ写す
	DeviceName   *string
	Status       OAuthDeviceAuthorizationStatus
	Interval     time.Duration
	ExpiresAt    time.Time
	LastPolledAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsExpiredは要求の有効期限が切れているかを返す
func (d *OAuthDeviceAuthorization) IsExpired(now time.Time) bool {
	return !now.Before(d.ExpiresAt)
}
