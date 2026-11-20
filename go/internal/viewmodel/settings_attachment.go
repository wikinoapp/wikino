package viewmodel

import (
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SettingsAttachmentは添付ファイルの管理の画面の一覧の1行に出す添付ファイルです
type SettingsAttachment struct {
	ID          string
	Filename    string
	ContentType string
	ByteSize    int64
	AttachedAt  time.Time
	// ThumbnailURLはサムネイルの署名付きURLです。画像でない添付ファイルと、オブジェクトストレージが
	// 設定されていないときは空にし、代わりにファイルのアイコンを出します
	ThumbnailURL string
}

// NewSettingsAttachmentsは、添付ファイルと、IDをキーにしたサムネイルの署名付きURLから一覧の行を生成します
func NewSettingsAttachments(attachments []*model.Attachment, thumbnailURLs map[model.AttachmentID]string) []SettingsAttachment {
	result := make([]SettingsAttachment, len(attachments))
	for i, attachment := range attachments {
		result[i] = SettingsAttachment{
			ID:           string(attachment.ID),
			Filename:     attachment.Filename,
			ContentType:  attachment.ContentType,
			ByteSize:     attachment.ByteSize,
			AttachedAt:   attachment.AttachedAt,
			ThumbnailURL: thumbnailURLs[attachment.ID],
		}
	}
	return result
}
