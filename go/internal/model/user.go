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

// Code returns the language tag of the locale, which is what i18n and the mail templates select a
// translation by.
//
// [Ja] Code はロケールの言語タグを返す。i18n とメールテンプレートは、この値で翻訳を選ぶ。
func (l Locale) Code() string {
	if l == LocaleEn {
		return "en"
	}
	return "ja"
}
