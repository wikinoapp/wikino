// Package timezone はタイムゾーンのコンテキスト操作を提供します
package timezone

import "context"

type contextKey string

const timezoneKey contextKey = "timezone"

// ToContext はコンテキストにタイムゾーンを設定する
func ToContext(ctx context.Context, tz string) context.Context {
	return context.WithValue(ctx, timezoneKey, tz)
}

// FromContext はコンテキストからタイムゾーン文字列を取得する
// タイムゾーンが設定されていない場合は "UTC" を返す
func FromContext(ctx context.Context) string {
	if tz, ok := ctx.Value(timezoneKey).(string); ok {
		return tz
	}
	return "UTC"
}
