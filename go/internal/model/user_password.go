package model

import (
	"time"
)

// UserPassword はユーザーパスワードのドメインモデル
type UserPassword struct {
	ID             string
	UserID         string
	PasswordDigest string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
