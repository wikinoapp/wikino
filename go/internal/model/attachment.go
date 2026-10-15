package model

import "time"

// AttachmentDownloadExpirationは、公開APIの添付ファイルの取得で返す署名付きURLの有効期間。
// Web画面の `/attachments/{ID}` (Rails版) がリダイレクトする署名付きURLの有効期間
// (ActiveStorageの既定値) と揃える。クライアントは受け取ってすぐに取得する前提で、URLが漏れたときに
// 使える時間を短くしておく
const AttachmentDownloadExpiration = 5 * time.Minute

// Attachmentは添付ファイルのドメインモデル
//
// 取得経路によってpopulateされるフィールドが異なる:
//   - FindByIDAndSpace / FindByIDsAndSpace: ID, SpaceID, Filename
//   - FindPubliclyReferencedBlobByID (公開og:image配信用): ID, SpaceID, BlobKey, ContentType
//   - FindBlobByIDAndSpace (公開APIの添付ファイルの取得用): ID, SpaceID, Filename, BlobKey, ContentType, ByteSize
//
// 取得経路とpopulate範囲はAttachmentRepositoryの各メソッドのdocを参照する。
type Attachment struct {
	ID          AttachmentID
	SpaceID     SpaceID
	Filename    string
	BlobKey     string
	ContentType string
	ByteSize    int64
}
