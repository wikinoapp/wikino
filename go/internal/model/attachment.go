package model

// Attachment は添付ファイルのドメインモデル
//
// 取得経路によって populate されるフィールドが異なる:
//   - FindByIDAndSpace / FindByIDsAndSpace: ID, SpaceID, Filename
//   - FindPubliclyReferencedBlobByID (公開 og:image 配信用): ID, SpaceID, BlobKey, ContentType
//
// 取得経路と populate 範囲は AttachmentRepository の各メソッドの doc を参照する。
type Attachment struct {
	ID          AttachmentID
	SpaceID     SpaceID
	Filename    string
	BlobKey     string
	ContentType string
}
