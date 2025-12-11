package model

import (
	"time"
)

// UserSession はユーザーセッションのドメインモデル
type UserSession struct {
	ID         string
	UserID     string
	Token      string
	IPAddress  string
	UserAgent  string
	SignedInAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
