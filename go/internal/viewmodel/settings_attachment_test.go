package viewmodel_test

import (
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewSettingsAttachments(t *testing.T) {
	t.Parallel()

	attachedAt := time.Date(2026, 11, 20, 1, 2, 3, 0, time.UTC)
	attachments := []*model.Attachment{
		{ID: "image-id", Filename: "photo.png", ContentType: "image/png", ByteSize: 2048, AttachedAt: attachedAt},
		{ID: "pdf-id", Filename: "doc.pdf", ContentType: "application/pdf", ByteSize: 10, AttachedAt: attachedAt},
	}
	got := viewmodel.NewSettingsAttachments(attachments, map[model.AttachmentID]string{"image-id": "https://example.com/signed"})

	if len(got) != 2 {
		t.Fatalf("len(got) = %d、期待値 = 2", len(got))
	}
	want := viewmodel.SettingsAttachment{
		ID: "image-id", Filename: "photo.png", ContentType: "image/png", ByteSize: 2048, AttachedAt: attachedAt,
		ThumbnailURL: "https://example.com/signed",
	}
	if got[0] != want {
		t.Errorf("got[0] = %+v、期待値 = %+v", got[0], want)
	}
	if got[1].ThumbnailURL != "" {
		t.Errorf("サムネイルの無い添付ファイルのThumbnailURL = %q、空を期待", got[1].ThumbnailURL)
	}
}
