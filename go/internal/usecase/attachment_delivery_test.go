package usecase

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

func TestAttachmentDeliveryOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		filename    string
		want        storage.PresignedGetOptions
	}{
		{
			name:        "画像はインラインで、記録された種類のまま返す",
			contentType: "image/png",
			filename:    "screenshot.png",
			want: storage.PresignedGetOptions{
				ResponseContentDisposition: `inline; filename="screenshot.png"; filename*=UTF-8''screenshot.png`,
				ResponseContentType:        "image/png",
			},
		},
		{
			name:        "PDFはインラインで返す",
			contentType: "application/pdf",
			filename:    "report.pdf",
			want: storage.PresignedGetOptions{
				ResponseContentDisposition: `inline; filename="report.pdf"; filename*=UTF-8''report.pdf`,
				ResponseContentType:        "application/pdf",
			},
		},
		{
			name:        "SVGはダウンロードさせ、種類をapplication/octet-streamに替える",
			contentType: "image/svg+xml",
			filename:    "diagram.svg",
			want: storage.PresignedGetOptions{
				ResponseContentDisposition: `attachment; filename="diagram.svg"; filename*=UTF-8''diagram.svg`,
				ResponseContentType:        "application/octet-stream",
			},
		},
		{
			name:        "HTMLはダウンロードさせ、種類をapplication/octet-streamに替える",
			contentType: "text/html",
			filename:    "page.html",
			want: storage.PresignedGetOptions{
				ResponseContentDisposition: `attachment; filename="page.html"; filename*=UTF-8''page.html`,
				ResponseContentType:        "application/octet-stream",
			},
		},
		{
			name:        "動画はダウンロードさせ、記録された種類のまま返す",
			contentType: "video/mp4",
			filename:    "movie.mp4",
			want: storage.PresignedGetOptions{
				ResponseContentDisposition: `attachment; filename="movie.mp4"; filename*=UTF-8''movie.mp4`,
				ResponseContentType:        "video/mp4",
			},
		},
		{
			name:        "日本語のファイル名は、filenameを「?」に、filename*をUTF-8のバイトにする",
			contentType: "image/jpeg",
			filename:    "写真.jpg",
			want: storage.PresignedGetOptions{
				ResponseContentDisposition: `inline; filename="%3F%3F.jpg"; filename*=UTF-8''%E5%86%99%E7%9C%9F.jpg`,
				ResponseContentType:        "image/jpeg",
			},
		},
		{
			name:        "アクセント記号の付いた文字は、filenameで記号を取り除く",
			contentType: "image/jpeg",
			filename:    "café.jpg",
			want: storage.PresignedGetOptions{
				ResponseContentDisposition: `inline; filename="cafe.jpg"; filename*=UTF-8''caf%C3%A9.jpg`,
				ResponseContentType:        "image/jpeg",
			},
		},
		{
			name:        "空白と引用符とパスの区切りを含むファイル名は、区切りと引用符を「-」にして空白をエスケープする",
			contentType: "application/zip",
			filename:    ` a "b"/c.zip `,
			want: storage.PresignedGetOptions{
				ResponseContentDisposition: `attachment; filename="a -b--c.zip"; filename*=UTF-8''a%20-b--c.zip`,
				ResponseContentType:        "application/zip",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := attachmentDeliveryOptions(&model.Attachment{ContentType: tt.contentType, Filename: tt.filename})
			if got != tt.want {
				t.Errorf("attachmentDeliveryOptions() = %+v、期待値 = %+v", got, tt.want)
			}
		})
	}
}
