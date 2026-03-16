package model

import "time"

// フィーチャーフラグ名の定数
// 新しいフィーチャーフラグを追加する場合は、ここに定数を追加する
const (
	FeatureFlagExample    FeatureFlagName = "go_example"
	FeatureFlagSuggestion FeatureFlagName = "go_suggestion"
)

// FeatureFlag はフィーチャーフラグのドメインモデル
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	UserID      *UserID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
