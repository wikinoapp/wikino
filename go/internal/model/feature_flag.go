package model

import "time"

// フィーチャーフラグ名の定数
// 新しいフィーチャーフラグを追加する場合は、ここに定数を追加する
// （FeatureFlagExample は命名規則の例として残している未使用の定数）
const (
	FeatureFlagExample FeatureFlagName = "go_example"
)

// FeatureFlag はフィーチャーフラグのドメインモデル
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	UserID      *UserID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
