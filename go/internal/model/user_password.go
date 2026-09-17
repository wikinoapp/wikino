package model

import (
	"time"
)

// UserPasswordはユーザーパスワードのドメインモデル
type UserPassword struct {
	ID             string
	UserID         UserID
	PasswordDigest string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
