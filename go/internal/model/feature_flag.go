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
	// FeatureFlagSpaceSettingsGeneralはスペースの一般設定 (/s/{space_identifier}/settings/general) をGo版で処理する
	FeatureFlagSpaceSettingsGeneral FeatureFlagName = "go_space_settings_general"
	// FeatureFlagSpaceAttachmentManagementはスペースの添付ファイルの管理 (/s/{space_identifier}/settings/attachments)
	// をGo版で処理する
	FeatureFlagSpaceAttachmentManagement FeatureFlagName = "go_space_attachment_management"
	// FeatureFlagSpaceDeletionはスペースの削除 (/s/{space_identifier}/settings/deletion) をGo版で処理する
	FeatureFlagSpaceDeletion FeatureFlagName = "go_space_deletion"
)

// AllFeatureFlagNamesは上で定義した全フラグの一覧。Goは定数グループの
// メンバーを列挙できないため、フラグを追加する人が必ず目にする定数のすぐ隣で
// 手作業で維持する。
var AllFeatureFlagNames = []FeatureFlagName{
	FeatureFlagExample,
	FeatureFlagPublicAPI,
	FeatureFlagRemoteMCPServer,
	FeatureFlagSpaceSettingsGeneral,
	FeatureFlagSpaceAttachmentManagement,
	FeatureFlagSpaceDeletion,
}

// FeatureFlagはフィーチャーフラグのドメインモデル
type FeatureFlag struct {
	ID          FeatureFlagID
	DeviceToken *string
	UserID      *UserID
	Name        FeatureFlagName
	CreatedAt   time.Time
}
