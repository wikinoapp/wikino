package model

import (
	"time"
)

// UserTwoFactorAuth はユーザーの二要素認証設定のドメインモデル
type UserTwoFactorAuth struct {
	ID            string
	UserID        UserID
	Secret        string
	Enabled       bool
	EnabledAt     *time.Time
	RecoveryCodes []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
