// Package model はドメインモデルを定義します
package model

import (
	"time"
)

// Locale はユーザーの言語設定を表す
type Locale int32

const (
	// LocaleJa は日本語
	LocaleJa Locale = 0
	// LocaleEn は英語
	LocaleEn Locale = 1
)

// User はユーザーのドメインモデル
type User struct {
	ID          UserID
	Email       string
	Atname      string
	Name        string
	Description string
	Locale      Locale
	TimeZone    string
	JoinedAt    time.Time
	DiscardedAt *time.Time
}
