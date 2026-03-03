package model

import "time"

// フィーチャーフラグ名の定数
const (
	FeatureFlagGoPageEdit FeatureFlagName = "go_page_edit"
)

// FeatureFlag はフィーチャーフラグのドメインモデル
type FeatureFlag struct {
	ID        FeatureFlagID
	UserID    UserID
	Name      FeatureFlagName
	CreatedAt time.Time
}
