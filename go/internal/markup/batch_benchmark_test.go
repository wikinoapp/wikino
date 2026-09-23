package markup

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// BenchmarkRenderHTMLは本文1件をRenderHTMLで表示用のHTMLにする時間を測る。
// Wikiリンクと添付ファイルを持つ本文では、レンダリング後の各工程が同じツリーを読み書きし、
// HTMLのパースが工程ごとに繰り返されないことをアロケーションで確かめる。時間は実行環境の
// 負荷でばらつくため、比較には使わない。
func BenchmarkRenderHTML(b *testing.B) {
	section := "## 見出し\n\n" +
		"本文の段落です。**強調** と `code` と [リンク](https://example.com/) を含みます。\n" +
		"改行を挟んだ2行目です。\n\n" +
		"- 項目1\n- 項目2\n  - 入れ子の項目\n- 項目3\n\n" +
		"```go\nfunc main() {\n\tprintln(\"hello\")\n}\n```\n\n"
	references := "[[ページA]] を参照してください。\n\n![写真](/attachments/att-1)\n\n"

	benchmarks := []struct {
		name string
		body string
	}{
		{name: "Markdownのみ", body: strings.Repeat(section, 5000/len(section)+1)},
		{name: "Wikiリンクと添付ファイルあり", body: strings.Repeat(section, 5000/len(section)+1) + references},
	}

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "ページA", TopicName: "topic", PageTitle: "ページA"},
				TopicName:  "topic",
				PageID:     model.PageID("page-1"),
				PageNumber: 1,
				PageTitle:  "ページA",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"},
		},
	}
	ctx := context.Background()

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(bm.body)))
			b.ResetTimer()
			for range b.N {
				if _, err := RenderHTML(ctx, bm.body, "topic", "space-1", "my-space", resolver, finder); err != nil {
					b.Fatalf("RenderHTML()のエラー = %v", err)
				}
			}
		})
	}
}
