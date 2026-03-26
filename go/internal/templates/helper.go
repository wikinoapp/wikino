// Package templates はHTMLテンプレート機能を提供します
package templates

import (
	"context"
	"time"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/timezone"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// ========================================
// templ用ヘルパー関数
// ========================================

// T は翻訳を取得する（templ用）
func T(ctx context.Context, messageID string, data ...map[string]any) string {
	return i18n.T(ctx, messageID, data...)
}

// Locale は現在のロケールを取得する
func Locale(ctx context.Context) string {
	return i18n.GetLocale(ctx)
}

// Deref はポインタを参照外しする（ジェネリック対応）
func Deref[T any](v *T) T {
	if v != nil {
		return *v
	}
	var zero T
	return zero
}

// ========================================
// 日時フォーマット関数
// ========================================

// FormatDateTime は日時を "2026/03/25 14:14" 形式でフォーマットする
func FormatDateTime(ctx context.Context, t time.Time) string {
	loc := loadLocationFromContext(ctx)
	return t.In(loc).Format("2006/01/02 15:04")
}

// FormatTime は時刻を "14:14" 形式でフォーマットする
func FormatTime(ctx context.Context, t time.Time) string {
	loc := loadLocationFromContext(ctx)
	return t.In(loc).Format("15:04")
}

// RelativeTime は相対時間文字列を返す
// 1分未満: "たった今"
// 1〜59分: "N分前"
// 1〜23時間: "N時間前"
// 1〜3日: "N日前"
// 3日超: 絶対時間にフォールバック（"2026/03/25 14:14"）
func RelativeTime(ctx context.Context, t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return T(ctx, "datetime_just_now")
	case d < time.Hour:
		minutes := int(d.Minutes())
		return T(ctx, "datetime_minutes_ago", map[string]any{"Count": minutes})
	case d < 24*time.Hour:
		hours := int(d.Hours())
		return T(ctx, "datetime_hours_ago", map[string]any{"Count": hours})
	case d < 72*time.Hour:
		days := int(d.Hours() / 24)
		return T(ctx, "datetime_days_ago", map[string]any{"Count": days})
	default:
		return FormatDateTime(ctx, t)
	}
}

// IsRelativeTime は指定された時刻が相対時間として表示されるかどうかを返す
func IsRelativeTime(t time.Time) bool {
	return time.Since(t) < 72*time.Hour
}

// loadLocationFromContext はコンテキストからタイムゾーンを取得し *time.Location を返す
func loadLocationFromContext(ctx context.Context) *time.Location {
	tz := timezone.FromContext(ctx)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

// ========================================
// アイコン関数
// ========================================

// Icon はアイコン名からSVGを返す（templ.Component対応）
// 可変長引数でクラス名を指定可能: Icon("name", "class1 class2")
func Icon(name viewmodel.IconName, class ...string) templ.Component {
	svg, ok := phosphorIcons[name]
	if !ok {
		svg, ok = customIcons[name]
	}
	if !ok {
		svg = phosphorIcons["info-regular"]
	}

	// クラス名が指定されている場合は、SVGタグに追加
	if len(class) > 0 && class[0] != "" {
		// <svg の直後にclass属性を挿入
		svg = `<svg class="` + class[0] + `" ` + svg[5:]
	}

	// templ.Rawを使用してSVGを返す
	return templ.Raw(svg)
}
