package model

// Attachmentは添付ファイルのドメインモデル
//
// 取得経路によってpopulateされるフィールドが異なる:
//   - FindByIDAndSpace / FindByIDsAndSpace: ID, SpaceID, Filename
//   - FindPubliclyReferencedBlobByID (公開og:image配信用): ID, SpaceID, BlobKey, ContentType
//
// 取得経路とpopulate範囲はAttachmentRepositoryの各メソッドのdocを参照する。
type Attachment struct {
	ID          AttachmentID
	SpaceID     SpaceID
	Filename    string
	BlobKey     string
	ContentType string
}
