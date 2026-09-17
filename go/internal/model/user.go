// Package modelはドメインモデルを定義します
package model

import (
	"time"
)

// Localeはユーザーの言語設定を表す
type Locale int32

const (
	// LocaleJaは日本語
	LocaleJa Locale = 0
	// LocaleEnは英語
	LocaleEn Locale = 1
)

// Userはユーザーのドメインモデル
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

// Codeはロケールの言語タグを返す。i18nとメールテンプレートは、この値で翻訳を選ぶ。
func (l Locale) Code() string {
	if l == LocaleEn {
		return "en"
	}
	return "ja"
}
