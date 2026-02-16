package model

// Attachment は添付ファイルのドメインモデル（存在確認・ファイル種別判定用）
type Attachment struct {
	ID       string
	SpaceID  SpaceID
	Filename string
}
