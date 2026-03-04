package model

import (
	"time"
)

// PasswordResetTokenExpirationDuration はパスワードリセットトークンの有効期限
const PasswordResetTokenExpirationDuration = 1 * time.Hour

// PasswordResetToken はパスワードリセットトークンのドメインモデル
type PasswordResetToken struct {
	ID          string
	UserID      UserID
	TokenDigest string
	ExpiresAt   time.Time
	UsedAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsExpired はトークンの有効期限が切れているかを返す
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed はトークンが使用済みかを返す
func (t *PasswordResetToken) IsUsed() bool {
	return t.UsedAt != nil
}

// IsValid はトークンが有効かを返す（未使用かつ有効期限内）
func (t *PasswordResetToken) IsValid() bool {
	return !t.IsUsed() && !t.IsExpired()
}
