// Package darkmodeは、ダークモードを適用するかの判定をコンテキストで受け渡す機能を提供します
package darkmode

import "context"

type contextKey string

const enabledKey contextKey = "dark_mode_enabled"

// ToContextは、ダークモードを適用するかを判定する関数をコンテキストに設定する。
// 判定はクエリを伴うため、値ではなく関数で持ち、ページ全体を描画するときにだけ呼ぶ
func ToContext(ctx context.Context, isEnabled func() bool) context.Context {
	return context.WithValue(ctx, enabledKey, isEnabled)
}

// IsEnabledは、コンテキストに設定された関数でダークモードを適用するかを判定する。
// 関数が設定されていない場合はfalseを返す
func IsEnabled(ctx context.Context) bool {
	if isEnabled, ok := ctx.Value(enabledKey).(func() bool); ok && isEnabled != nil {
		return isEnabled()
	}
	return false
}
