package model

import "time"

// Feature flag names. When introducing a new flag, add a constant here and add
// it to AllFeatureFlagNames as well.
// (FeatureFlagExample is an unused constant kept as an example of the naming.)
//
// [Ja] フィーチャーフラグ名の定数。新しいフィーチャーフラグを追加する場合は、
// ここに定数を追加し、AllFeatureFlagNames にも追加する。
// (FeatureFlagExample は命名規則の例として残している未使用の定数)
const (
	FeatureFlagExample FeatureFlagName = "go_example"
)

// AllFeatureFlagNames lists every flag defined above. Go cannot enumerate the
// members of a constant group, so the list is kept by hand right next to the
// constants, where whoever adds a flag is already looking.
//
// [Ja] AllFeatureFlagNames は上で定義した全フラグの一覧。Go は定数グループの
// メンバーを列挙できないため、フラグを追加する人が必ず目にする定数のすぐ隣で
// 手作業で維持する。
var AllFeatureFlagNames = []FeatureFlagName{
	FeatureFlagExample,
}

// FeatureFlag はフィーチャーフラグのドメインモデル
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	UserID      *UserID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
