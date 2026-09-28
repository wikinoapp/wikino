package model

import "time"

// フィーチャーフラグ名の定数。新しいフィーチャーフラグを追加する場合は、
// ここに定数を追加し、AllFeatureFlagNamesにも追加する。
// (FeatureFlagExampleは命名規則の例として残している未使用の定数)
const (
	FeatureFlagExample FeatureFlagName = "go_example"
	// FeatureFlagPublicAPIは公開Web API (個人アクセストークン・OAuth連携・APIの呼び出し) を有効にする
	FeatureFlagPublicAPI FeatureFlagName = "go_public_api"
)

// AllFeatureFlagNamesは上で定義した全フラグの一覧。Goは定数グループの
// メンバーを列挙できないため、フラグを追加する人が必ず目にする定数のすぐ隣で
// 手作業で維持する。
var AllFeatureFlagNames = []FeatureFlagName{
	FeatureFlagExample,
	FeatureFlagPublicAPI,
}

// FeatureFlagはフィーチャーフラグのドメインモデル
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	UserID      *UserID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
