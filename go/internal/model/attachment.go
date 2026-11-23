package model

import (
	"slices"
	"strings"
	"time"
)

// AttachmentDownloadExpirationは、公開APIの添付ファイルの取得で返す署名付きURLと、Web画面の
// `/attachments/{ID}` がリダイレクトする署名付きURLの有効期間。クライアントは受け取ってすぐに取得する前提で、URLが漏れたときに
// 使える時間を短くしておく
const AttachmentDownloadExpiration = 5 * time.Minute

// AttachmentSignedURLExpirationは、Web画面のページ本文に添付ファイルを表示するために一括で返す
// 署名付きURL (`POST /attachments/signed_urls`) の有効期間。画面を開いている間に
// 遅延読み込みの画像が取得されるため、リダイレクト先のURLより長くする
const AttachmentSignedURLExpiration = time.Hour

// AttachmentUploadURLExpirationは、事前署名で返すアップロードの署名付きURLの有効期間
const AttachmentUploadURLExpiration = 5 * time.Minute

// AttachmentBlobServiceNameは、添付ファイルのblobのservice_nameに書く値。Rails期に作られた行と同じ値にし、
// どの行の実体も同じバケットにあることを表す
const AttachmentBlobServiceName = "cloudflare_r2"

// AttachmentUploadDeadlineは、事前署名した添付ファイルのアップロードを待つ上限の時間。アップロード後の
// 処理は、事前署名からこの時間を過ぎてもオブジェクトが無ければ、失敗として記録する。署名付きURLの
// 有効期限より長くするのは、期限の前に始まったアップロードが、遅い回線では期限のあとも続くためである
const AttachmentUploadDeadline = time.Hour

// AttachmentProcessingStatusは、アップロード後の処理の状態。値はattachments.processing_statusの列に
// 保存する整数で、Rails期に作られた行と共通にする
type AttachmentProcessingStatus int32

const (
	// AttachmentProcessingStatusPendingは、処理を待っている状態
	AttachmentProcessingStatusPending AttachmentProcessingStatus = 0
	// AttachmentProcessingStatusProcessingは、処理している状態
	AttachmentProcessingStatusProcessing AttachmentProcessingStatus = 1
	// AttachmentProcessingStatusCompletedは、処理を終えた状態
	AttachmentProcessingStatusCompleted AttachmentProcessingStatus = 2
	// AttachmentProcessingStatusFailedは、処理できなかった状態。アップロードされなかったときと、
	// 内容を処理できなかったときにこの状態にする
	AttachmentProcessingStatusFailed AttachmentProcessingStatus = 3
)

// attachmentImageMaxByteSize・attachmentVideoMaxByteSize・attachmentOtherMaxByteSizeは、種類ごとの
// 添付ファイルの大きさの上限
const (
	attachmentImageMaxByteSize = 10 * 1024 * 1024
	attachmentVideoMaxByteSize = 100 * 1024 * 1024
	attachmentOtherMaxByteSize = 25 * 1024 * 1024
)

// attachmentAllowedContentTypesは、アップロードを許可する種類
var attachmentAllowedContentTypes = []string{
	// 画像
	"image/jpeg",
	"image/jpg",
	"image/png",
	"image/gif",
	"image/svg+xml",
	"image/webp",
	// 動画
	"video/mp4",
	"video/webm",
	"video/quicktime",
	// ドキュメント
	"application/pdf",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.ms-excel",
	// テキストファイル
	"text/plain",
	"text/csv",
	"text/markdown",
	"application/json",
	// アーカイブ
	"application/zip",
	"application/x-zip-compressed",
	"application/x-tar",
	"application/gzip",
	"application/x-gzip",
	"application/x-compressed-tar",
}

// IsAttachmentContentTypeAllowedは、種類contentTypeのアップロードを許可するかどうかを返す
func IsAttachmentContentTypeAllowed(contentType string) bool {
	return slices.Contains(attachmentAllowedContentTypes, contentType)
}

// AttachmentMaxByteSizeは、種類contentTypeの添付ファイルの大きさの上限を返す
func AttachmentMaxByteSize(contentType string) int64 {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return attachmentImageMaxByteSize
	case strings.HasPrefix(contentType, "video/"):
		return attachmentVideoMaxByteSize
	default:
		return attachmentOtherMaxByteSize
	}
}

// Attachmentは添付ファイルのドメインモデル
//
// 取得経路によってpopulateされるフィールドが異なる:
//   - FindByIDAndSpace / FindByIDsAndSpace: ID, SpaceID, Filename
//   - FindPubliclyReferencedBlobByID (公開og:image配信用): ID, SpaceID, BlobKey, ContentType
//   - FindPubliclyReferencedBlobByIDAndSpace (所属スペースでの公開判定用): ID, SpaceID, BlobKey, ContentType
//   - FindBlobByIDAndSpace (公開APIの添付ファイルの取得用): ID, SpaceID, Filename, BlobKey, ContentType, ByteSize
//   - FindBlobByID (Web画面の添付ファイルの表示用): ID, SpaceID, Filename, BlobKey, ContentType, ByteSize
//   - FindBlobsByIDs (Web画面の署名付きURLの一括取得用): ID, SpaceID, Filename, BlobKey, ContentType, ByteSize
//   - FindForProcessingByIDAndSpace (アップロード後の処理用): ID, SpaceID, BlobKey, ContentType, ByteSize, ProcessingStatus, AttachedAt
//   - ListBySpacePaginated (添付ファイルの管理の画面の一覧用): ID, SpaceID, Filename, BlobKey, ContentType, ByteSize, AttachedAt
//
// 取得経路とpopulate範囲はAttachmentRepositoryの各メソッドのdocを参照する。
type Attachment struct {
	ID          AttachmentID
	SpaceID     SpaceID
	Filename    string
	BlobKey     string
	ContentType string
	ByteSize    int64
	// ProcessingStatusは、アップロード後の処理の状態
	ProcessingStatus AttachmentProcessingStatus
	// AttachedAtは、添付ファイルを作った (事前署名した) 時刻
	AttachedAt time.Time
}

// IsImageは、添付ファイルが画像かどうかを種類から返す。種類が
// `image/` で始まるものを画像とする
func (a *Attachment) IsImage() bool {
	return strings.HasPrefix(a.ContentType, "image/")
}
