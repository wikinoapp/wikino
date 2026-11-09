package markup

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

const (
	testAppURL  = "https://wikino.example.com"
	testPageURL = "https://wikino.example.com/s/my-space/pages/7"
)

func TestAbsolutizeURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "空の本文は空のまま返す",
			html: "",
			want: "",
		},
		{
			name: "アプリ内の相対リンクにアプリのURLを付ける",
			html: `<p><a href="/s/my-space/pages/3">ページ</a></p>`,
			want: `<p><a href="https://wikino.example.com/s/my-space/pages/3">ページ</a></p>`,
		},
		{
			name: "アプリ内の相対パスの画像にアプリのURLを付ける",
			html: `<p><img src="/images/logo.png" alt="logo"/></p>`,
			want: `<p><img src="https://wikino.example.com/images/logo.png" alt="logo"/></p>`,
		},
		{
			name: "ページ内のリンクをページの絶対URLの後ろに付ける",
			html: `<p><a href="#見出し">見出しへ</a></p>`,
			want: `<p><a href="https://wikino.example.com/s/my-space/pages/7#見出し">見出しへ</a></p>`,
		},
		{
			name: "外部リンクは変えない",
			html: `<p><a href="https://example.org/a" rel="nofollow">外部</a></p>`,
			want: `<p><a href="https://example.org/a" rel="nofollow">外部</a></p>`,
		},
		{
			name: "スキームを省いたURLは外部として変えない",
			html: `<p><a href="//example.org/a">外部</a></p>`,
			want: `<p><a href="//example.org/a">外部</a></p>`,
		},
		{
			name: "メールのリンクは変えない",
			html: `<p><a href="mailto:a@example.org">メール</a></p>`,
			want: `<p><a href="mailto:a@example.org">メール</a></p>`,
		},
		{
			name: "添付ファイルのプレースホルダーを添付ファイルのURLにし、描画用の属性を外す",
			html: `<p><a href="#" data-attachment-id="att-1" data-attachment-link="true" target="_blank">doc.pdf</a></p>`,
			want: `<p><a href="https://wikino.example.com/attachments/att-1" target="_blank">doc.pdf</a></p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := AbsolutizeURLs(tt.html, testAppURL, testPageURL)
			if err != nil {
				t.Fatalf("AbsolutizeURLs()のエラー = %v", err)
			}
			if got != tt.want {
				t.Errorf("AbsolutizeURLs() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestAbsolutizeURLs_TrailingSlashOfAppURL(t *testing.T) {
	t.Parallel()

	got, err := AbsolutizeURLs(`<a href="/s/my-space">スペース</a>`, testAppURL+"/", testPageURL)
	if err != nil {
		t.Fatalf("AbsolutizeURLs()のエラー = %v", err)
	}
	want := `<a href="https://wikino.example.com/s/my-space">スペース</a>`
	if got != want {
		t.Errorf("AbsolutizeURLs() = %q、期待値 = %q", got, want)
	}
}

// TestAbsolutizeURLs_RenderedBodyは、RenderHTMLBatchが描画した本文を書き換えたときに、
// 画面のJSに頼る部分が残らないことを確かめる。
func TestAbsolutizeURLs_RenderedBody(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "別のページ", TopicName: "topic1", PageTitle: "別のページ"},
				TopicName:  "topic1",
				PageID:     "page-3",
				PageNumber: 3,
				PageTitle:  "別のページ",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-image", SpaceID: "space-1", Filename: "photo.png"},
			{ID: "att-video", SpaceID: "space-1", Filename: "movie.mp4"},
			{ID: "att-doc", SpaceID: "space-1", Filename: "doc.pdf"},
			{ID: "att-zip", SpaceID: "space-1", Filename: "archive.zip"},
		},
	}
	body := strings.Join([]string{
		"## 見出し",
		"",
		"[[別のページ]]",
		"",
		"![写真](/attachments/att-image)",
		"",
		"[動画](/attachments/att-video)",
		"",
		"[資料](/attachments/att-doc)",
		"",
		"![圧縮ファイル](/attachments/att-zip)",
		"",
		"[見出しへ](#見出し) [外部](https://example.org/a)",
	}, "\n")

	rendered, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("RenderHTML()のエラー = %v", err)
	}

	got, err := AbsolutizeURLs(rendered, testAppURL, testPageURL)
	if err != nil {
		t.Fatalf("AbsolutizeURLs()のエラー = %v", err)
	}

	wantContains := []string{
		`href="https://wikino.example.com/s/my-space/pages/3"`,
		`href="https://wikino.example.com/attachments/att-image"`,
		`<img src="https://wikino.example.com/attachments/att-image"`,
		`<video src="https://wikino.example.com/attachments/att-video"`,
		`<a href="https://wikino.example.com/attachments/att-video"`,
		`<a href="https://wikino.example.com/attachments/att-doc"`,
		`href="https://wikino.example.com/attachments/att-zip"`,
		// Markdownのリンク先はレンダラーがパーセントエンコードする
		`href="https://wikino.example.com/s/my-space/pages/7#%E8%A6%8B%E5%87%BA%E3%81%97"`,
		`href="https://example.org/a"`,
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("結果に %q が含まれていない: %s", want, got)
		}
	}

	wantNotContains := []string{
		"data-attachment-",
		`href="#"`,
		`src=""`,
		`href="/`,
		`src="/`,
	}
	for _, notWant := range wantNotContains {
		if strings.Contains(got, notWant) {
			t.Errorf("結果に %q が残っている: %s", notWant, got)
		}
	}
}
