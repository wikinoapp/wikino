package model

import "time"

// フィーチャーフラグ名の定数
// 新しいフィーチャーフラグを追加する場合は、ここに定数を追加する
// （FeatureFlagExample は命名規則の例として残している未使用の定数）
const (
	FeatureFlagExample FeatureFlagName = "go_example"

	// FeatureFlagPageShow gates the Go version of the page detail screen
	// (GET /s/:space_identifier/pages/:page_number).
	//
	// [Ja] ページ表示画面 (GET /s/:space_identifier/pages/:page_number) の
	// Go 版表示を制御する。
	FeatureFlagPageShow FeatureFlagName = "go_page_show"
)

// FeatureFlag はフィーチャーフラグのドメインモデル
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	UserID      *UserID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
