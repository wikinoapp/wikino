package model

import "time"

// フィーチャーフラグ名の定数。新しいフィーチャーフラグを追加する場合は、
// ここに定数を追加し、AllFeatureFlagNamesにも追加する。
// (FeatureFlagExampleは命名規則の例として残している未使用の定数)
const (
	FeatureFlagExample FeatureFlagName = "go_example"
	// FeatureFlagPublicAPIは公開Web API (個人用アクセストークン・OAuth連携・APIの呼び出し) を有効にする
	FeatureFlagPublicAPI FeatureFlagName = "go_public_api"
	// FeatureFlagRemoteMCPServerはリモートMCPサーバー (MCPクライアントとの連携・ツールの呼び出し) を有効にする
	FeatureFlagRemoteMCPServer FeatureFlagName = "go_remote_mcp_server"
	// FeatureFlagDarkModeはGo版の画面のダークモード (OSの配色がダークの閲覧者への暗い配色) を有効にする
	FeatureFlagDarkMode FeatureFlagName = "go_dark_mode"
	// FeatureFlagProfileはプロフィール (/@{atname}) をGo版で処理する
	FeatureFlagProfile FeatureFlagName = "go_profile"
	// FeatureFlagUserSettingsはユーザー設定のトップ・プロフィールの編集・メールアドレスの変更をGo版で処理する
	FeatureFlagUserSettings FeatureFlagName = "go_user_settings"
	// FeatureFlagTwoFactorAuthSettingsは二要素認証の設定 (/settings/two_factor_authとその配下) をGo版で処理する
	FeatureFlagTwoFactorAuthSettings FeatureFlagName = "go_two_factor_auth_settings"
	// FeatureFlagAccountDeletionはアカウントの削除 (/settings/account/deletionとその確認画面) をGo版で処理する
	FeatureFlagAccountDeletion FeatureFlagName = "go_account_deletion"
)

// AllFeatureFlagNamesは上で定義した全フラグの一覧。Goは定数グループの
// メンバーを列挙できないため、フラグを追加する人が必ず目にする定数のすぐ隣で
// 手作業で維持する。
var AllFeatureFlagNames = []FeatureFlagName{
	FeatureFlagExample,
	FeatureFlagPublicAPI,
	FeatureFlagRemoteMCPServer,
	FeatureFlagDarkMode,
	FeatureFlagProfile,
	FeatureFlagUserSettings,
	FeatureFlagTwoFactorAuthSettings,
	FeatureFlagAccountDeletion,
}

// FeatureFlagはフィーチャーフラグのドメインモデル
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	UserID      *UserID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
