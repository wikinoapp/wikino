package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// AttachmentBuilder は添付ファイルテストデータのビルダー
type AttachmentBuilder struct {
	t  *testing.T
	tx *sql.Tx

	spaceID       string
	spaceMemberID string
	filename      string
	contentType   string
	byteSize      int64
}

// NewAttachmentBuilder は AttachmentBuilder を生成します
func NewAttachmentBuilder(t *testing.T, tx *sql.Tx) *AttachmentBuilder {
	t.Helper()
	return &AttachmentBuilder{
		t:           t,
		tx:          tx,
		filename:    "test.png",
		contentType: "image/png",
		byteSize:    1024,
	}
}

// WithSpaceID はスペースIDを設定します
func (b *AttachmentBuilder) WithSpaceID(spaceID model.SpaceID) *AttachmentBuilder {
	b.spaceID = string(spaceID)
	return b
}

// WithSpaceMemberID はスペースメンバーIDを設定します
func (b *AttachmentBuilder) WithSpaceMemberID(spaceMemberID model.SpaceMemberID) *AttachmentBuilder {
	b.spaceMemberID = string(spaceMemberID)
	return b
}

// WithFilename はファイル名を設定します
func (b *AttachmentBuilder) WithFilename(filename string) *AttachmentBuilder {
	b.filename = filename
	return b
}

// WithContentType はコンテンツタイプを設定します
func (b *AttachmentBuilder) WithContentType(contentType string) *AttachmentBuilder {
	b.contentType = contentType
	return b
}

// Build は添付ファイルを作成し、IDを返します
func (b *AttachmentBuilder) Build() model.AttachmentID {
	b.t.Helper()

	if b.spaceID == "" {
		b.t.Fatal("AttachmentBuilder: spaceIDが設定されていません。WithSpaceID()を呼んでください")
	}
	if b.spaceMemberID == "" {
		b.t.Fatal("AttachmentBuilder: spaceMemberIDが設定されていません。WithSpaceMemberID()を呼んでください")
	}

	now := time.Now()

	// active_storage_blobを作成
	var blobID string
	err := b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO active_storage_blobs (key, filename, content_type, service_name, byte_size, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		"test-key-"+now.Format("20060102150405.000000000"), b.filename, b.contentType, "local", b.byteSize, now,
	).Scan(&blobID)
	if err != nil {
		b.t.Fatalf("active_storage_blob作成に失敗: %v", err)
	}

	// active_storage_attachmentを作成
	var attachmentStorageID string
	err = b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		"file", "Space", b.spaceID, blobID, now,
	).Scan(&attachmentStorageID)
	if err != nil {
		b.t.Fatalf("active_storage_attachment作成に失敗: %v", err)
	}

	// attachmentを作成
	var attachmentID string
	err = b.tx.QueryRowContext(
		context.Background(),
		`INSERT INTO attachments (space_id, active_storage_attachment_id, attached_space_member_id, attached_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		b.spaceID, attachmentStorageID, b.spaceMemberID, now, now, now,
	).Scan(&attachmentID)
	if err != nil {
		b.t.Fatalf("attachment作成に失敗: %v", err)
	}

	return model.AttachmentID(attachmentID)
}
