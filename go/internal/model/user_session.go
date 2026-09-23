package model

import (
	"time"
)

// UserSessionはユーザーセッションのドメインモデル
type UserSession struct {
	ID         string
	UserID     UserID
	Token      string
	IPAddress  string
	UserAgent  string
	SignedInAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
