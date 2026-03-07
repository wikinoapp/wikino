package markup

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestRenderHTML_TaskListWithWikilinks(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "設計メモ", TopicName: "開発", PageTitle: "設計メモ"},
				TopicName:  "開発",
				PageID:     model.PageID("page-1"),
				PageNumber: 1,
				PageTitle:  "設計メモ",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{}

	body := "- [ ] [[設計メモ]]を確認する\n- [x] レビュー完了"
	got, err := RenderHTML(context.Background(), body, "開発", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// タスクリストのチェックボックスが含まれること
	if !strings.Contains(got, `type="checkbox"`) {
		t.Errorf("result should contain task list checkboxes, got: %s", got)
	}
	// Wikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("result should contain wikilink, got: %s", got)
	}
	if !strings.Contains(got, "設計メモ</a>") {
		t.Errorf("result should contain page title in link, got: %s", got)
	}
	// チェック済みタスクの存在
	if !strings.Contains(got, `checked`) {
		t.Errorf("result should contain checked checkbox, got: %s", got)
	}
}

func TestRenderHTML_MultipleParagraphsWithWikilinksAndAttachments(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "API設計", TopicName: "開発", PageTitle: "API設計"},
				TopicName:  "開発",
				PageID:     model.PageID("page-1"),
				PageNumber: 10,
				PageTitle:  "API設計",
			},
			{
				Key:        WikilinkKey{Raw: "設計/DB設計", TopicName: "設計", PageTitle: "DB設計"},
				TopicName:  "設計",
				PageID:     model.PageID("page-2"),
				PageNumber: 20,
				PageTitle:  "DB設計",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-img", SpaceID: "space-1", Filename: "diagram.png"},
			{ID: "att-doc", SpaceID: "space-1", Filename: "spec.pdf"},
		},
	}

	body := `[[API設計]]を参照してください。

![構成図](/attachments/att-img)

詳細は[[設計/DB設計]]と[仕様書](/attachments/att-doc)にあります。`

	got, err := RenderHTML(context.Background(), body, "開発", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 同一トピックのWikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/10"`) {
		t.Errorf("result should contain same-topic wikilink, got: %s", got)
	}
	// クロストピックのWikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/20"`) {
		t.Errorf("result should contain cross-topic wikilink, got: %s", got)
	}
	// 画像添付ファイルが変換されていること
	if !strings.Contains(got, `data-attachment-id="att-img"`) {
		t.Errorf("result should contain image attachment, got: %s", got)
	}
	if !strings.Contains(got, `data-attachment-type="image"`) {
		t.Errorf("result should contain image type attribute, got: %s", got)
	}
	// PDF添付ファイルがdata属性付きリンクに変換されていること
	if !strings.Contains(got, `data-attachment-id="att-doc"`) {
		t.Errorf("result should contain PDF attachment, got: %s", got)
	}
	// Markdownリンク記法のテキスト「仕様書」が保持されていること
	if !strings.Contains(got, "仕様書</a>") {
		t.Errorf("result should preserve original link text, got: %s", got)
	}
}

func TestRenderHTML_WikilinkInCodeBlockNotConverted(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "ページA", TopicName: "topic1", PageTitle: "ページA"},
				TopicName:  "topic1",
				PageID:     model.PageID("page-1"),
				PageNumber: 1,
				PageTitle:  "ページA",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{}

	body := "通常テキスト: [[ページA]]\n\n```\nコード内: [[ページA]]\n```"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 通常テキスト内のWikiリンクは変換される
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("result should contain wikilink in normal text, got: %s", got)
	}
	// コードブロック内のWikiリンクは変換されない（[[ページA]]がそのまま残る）
	if !strings.Contains(got, "<code>") {
		t.Errorf("result should contain code block, got: %s", got)
	}
}

func TestRenderHTML_NonExistentPageWikilinkAsPlainText(t *testing.T) {
	t.Parallel()

	// ページが存在しないのでresolverは空のロケーションを返す
	resolver := &mockPageLocationResolver{
		locations: []PageLocation{},
	}
	finder := &mockBatchAttachmentFinder{}

	body := "参照: [[存在しないページ]]"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// リンクタグに変換されないこと
	if strings.Contains(got, `<a href="/s/my-space/pages/`) {
		t.Errorf("non-existent page should not become a link, got: %s", got)
	}
	// 元のテキストが残ること
	if !strings.Contains(got, "存在しないページ") {
		t.Errorf("page title text should remain, got: %s", got)
	}
}

func TestRenderHTML_NonExistentAttachmentNotTransformed(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	// 添付ファイルが存在しないのでfinderは空を返す
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{},
	}

	body := "画像: ![alt](/attachments/nonexistent-id)"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// data-attachment-id属性が付与されないこと
	if strings.Contains(got, `data-attachment-id`) {
		t.Errorf("non-existent attachment should not be transformed, got: %s", got)
	}
	// img要素自体は残ること（サニタイズされた状態で）
	if !strings.Contains(got, "img") {
		t.Errorf("img element should remain, got: %s", got)
	}
}

func TestRenderHTML_SpecialCharactersInWikilinkTitle(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "日記 (2025)", TopicName: "topic1", PageTitle: "日記 (2025)"},
				TopicName:  "topic1",
				PageID:     model.PageID("page-1"),
				PageNumber: 5,
				PageTitle:  "日記 (2025)",
			},
			{
				Key:        WikilinkKey{Raw: "Notebook -> List", TopicName: "topic1", PageTitle: "Notebook -> List"},
				TopicName:  "topic1",
				PageID:     model.PageID("page-2"),
				PageNumber: 6,
				PageTitle:  "Notebook -> List",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{}

	body := "[[日記 (2025)]]と[[Notebook -> List]]を参照"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 括弧を含むタイトルが正しくリンクされること
	if !strings.Contains(got, `<a href="/s/my-space/pages/5"`) {
		t.Errorf("result should contain link for parenthesized title, got: %s", got)
	}
	// 矢印を含むタイトルが正しくリンクされること
	if !strings.Contains(got, `<a href="/s/my-space/pages/6"`) {
		t.Errorf("result should contain link for arrow title, got: %s", got)
	}
}

func TestRenderHTML_MultipleAttachmentTypes(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-jpg", SpaceID: "space-1", Filename: "photo.jpg"},
			{ID: "att-png", SpaceID: "space-1", Filename: "screenshot.png"},
			{ID: "att-pdf", SpaceID: "space-1", Filename: "document.pdf"},
			{ID: "att-mp4", SpaceID: "space-1", Filename: "demo.mp4"},
		},
	}

	body := `![写真](/attachments/att-jpg)

![スクショ](/attachments/att-png)

![ドキュメント](/attachments/att-pdf)

[デモ動画](/attachments/att-mp4)`

	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// JPG画像: インライン画像として変換
	if !strings.Contains(got, `data-attachment-id="att-jpg"`) {
		t.Errorf("result should contain JPG attachment, got: %s", got)
	}
	// PNG画像: インライン画像として変換
	if !strings.Contains(got, `data-attachment-id="att-png"`) {
		t.Errorf("result should contain PNG attachment, got: %s", got)
	}
	// PDF: ダウンロードリンクとして変換（ファイル名表示）
	if !strings.Contains(got, `data-attachment-id="att-pdf"`) {
		t.Errorf("result should contain PDF attachment, got: %s", got)
	}
	if !strings.Contains(got, "document.pdf") {
		t.Errorf("result should show PDF filename, got: %s", got)
	}
	// MP4動画: video要素として変換
	if !strings.Contains(got, `data-attachment-id="att-mp4"`) {
		t.Errorf("result should contain MP4 attachment, got: %s", got)
	}
	if !strings.Contains(got, "<video") {
		t.Errorf("result should contain video element for MP4, got: %s", got)
	}
}

func TestRenderHTML_ComplexDocument(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "要件定義", TopicName: "プロジェクト", PageTitle: "要件定義"},
				TopicName:  "プロジェクト",
				PageID:     model.PageID("page-1"),
				PageNumber: 1,
				PageTitle:  "要件定義",
			},
			{
				Key:        WikilinkKey{Raw: "設計/テーブル設計", TopicName: "設計", PageTitle: "テーブル設計"},
				TopicName:  "設計",
				PageID:     model.PageID("page-2"),
				PageNumber: 2,
				PageTitle:  "テーブル設計",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-arch", SpaceID: "space-1", Filename: "architecture.png"},
			{ID: "att-spec", SpaceID: "space-1", Filename: "spec.xlsx"},
		},
	}

	body := `# プロジェクト概要

このドキュメントは**重要な**設計資料です。

## 参考資料

- [[要件定義]]を参照
- [[設計/テーブル設計]]も確認

## アーキテクチャ図

![アーキテクチャ](/attachments/att-arch)

## タスク

- [ ] [[要件定義]]のレビュー
- [x] 仕様書の確認: [仕様書](/attachments/att-spec)
- [ ] コードレビュー`

	got, err := RenderHTML(context.Background(), body, "プロジェクト", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 見出しが変換されていること
	if !strings.Contains(got, "<h1") {
		t.Errorf("result should contain h1 heading, got: %s", got)
	}
	if !strings.Contains(got, "<h2") {
		t.Errorf("result should contain h2 headings, got: %s", got)
	}
	// 太字が変換されていること
	if !strings.Contains(got, "<strong>重要な</strong>") {
		t.Errorf("result should contain bold text, got: %s", got)
	}
	// Wikiリンクが変換されていること（リスト内、タスクリスト内の両方）
	if count := strings.Count(got, `href="/s/my-space/pages/1"`); count < 2 {
		t.Errorf("result should contain at least 2 links to page 1, got %d in: %s", count, got)
	}
	if !strings.Contains(got, `<a href="/s/my-space/pages/2"`) {
		t.Errorf("result should contain cross-topic wikilink, got: %s", got)
	}
	// 画像添付ファイルが変換されていること
	if !strings.Contains(got, `data-attachment-id="att-arch"`) {
		t.Errorf("result should contain architecture image, got: %s", got)
	}
	// Excel添付ファイルがリンクに変換されていること
	if !strings.Contains(got, `data-attachment-id="att-spec"`) {
		t.Errorf("result should contain spec attachment, got: %s", got)
	}
	// タスクリストが含まれること
	if !strings.Contains(got, `type="checkbox"`) {
		t.Errorf("result should contain task list checkboxes, got: %s", got)
	}
}

func TestRenderHTML_StandaloneImageGetsWrappedInParagraph(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"},
		},
	}

	body := "テキスト\n\n![画像](/attachments/att-1)\n\nテキスト2"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 画像リンクが存在すること
	if !strings.Contains(got, `class="wikino-attachment-image-link"`) {
		t.Errorf("result should contain image link class, got: %s", got)
	}
	// スタンドアロン画像が<p>でラップされていること
	if !strings.Contains(got, `<p><a`) {
		t.Errorf("standalone image should be wrapped in <p>, got: %s", got)
	}
}

func TestRenderHTML_MultipleWikilinksOnSameLine(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "ページA", TopicName: "topic1", PageTitle: "ページA"},
				TopicName:  "topic1",
				PageID:     model.PageID("page-1"),
				PageNumber: 1,
				PageTitle:  "ページA",
			},
			{
				Key:        WikilinkKey{Raw: "topic2/ページB", TopicName: "topic2", PageTitle: "ページB"},
				TopicName:  "topic2",
				PageID:     model.PageID("page-2"),
				PageNumber: 2,
				PageTitle:  "ページB",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{}

	body := "[[ページA]]と[[topic2/ページB]]の両方を参照してください。"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 両方のWikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("result should contain first wikilink, got: %s", got)
	}
	if !strings.Contains(got, `<a href="/s/my-space/pages/2"`) {
		t.Errorf("result should contain second wikilink, got: %s", got)
	}
	if !strings.Contains(got, "ページA</a>") {
		t.Errorf("result should contain first page title, got: %s", got)
	}
	if !strings.Contains(got, "ページB</a>") {
		t.Errorf("result should contain second page title, got: %s", got)
	}
}

func TestRenderHTML_GFMTableWithWikilink(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "詳細ページ", TopicName: "topic1", PageTitle: "詳細ページ"},
				TopicName:  "topic1",
				PageID:     model.PageID("page-1"),
				PageNumber: 3,
				PageTitle:  "詳細ページ",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{}

	body := "| 項目 | 説明 |\n| --- | --- |\n| 参照先 | [[詳細ページ]] |"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// テーブルが生成されていること
	if !strings.Contains(got, "<table") {
		t.Errorf("result should contain table, got: %s", got)
	}
	// テーブル内のWikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/3"`) {
		t.Errorf("result should contain wikilink in table, got: %s", got)
	}
}

func TestRenderHTML_MixedExistentAndNonExistentWikilinks(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "存在するページ", TopicName: "topic1", PageTitle: "存在するページ"},
				TopicName:  "topic1",
				PageID:     model.PageID("page-1"),
				PageNumber: 1,
				PageTitle:  "存在するページ",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{}

	body := "[[存在するページ]]と[[存在しないページ]]があります。"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 存在するページはリンクに変換される
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("existing page should become a link, got: %s", got)
	}
	// 存在しないページはリンクに変換されない
	linkCount := strings.Count(got, `<a href="/s/my-space/pages/`)
	if linkCount != 1 {
		t.Errorf("expected exactly 1 wikilink, got %d in: %s", linkCount, got)
	}
	// 存在しないページのテキストは残る
	if !strings.Contains(got, "存在しないページ") {
		t.Errorf("non-existent page title should remain as text, got: %s", got)
	}
}

func TestRenderHTML_InlineImageWithSurroundingText(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "icon.png"},
		},
	}

	body := "テキストの中に![アイコン](/attachments/att-1)が含まれています。"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 添付ファイルが変換されていること
	if !strings.Contains(got, `data-attachment-id="att-1"`) {
		t.Errorf("result should contain attachment, got: %s", got)
	}
	// 前後のテキストが保持されていること
	if !strings.Contains(got, "テキストの中に") {
		t.Errorf("surrounding text should be preserved, got: %s", got)
	}
	if !strings.Contains(got, "が含まれています") {
		t.Errorf("surrounding text should be preserved, got: %s", got)
	}
}

func TestRenderHTML_HTMLImgWithCaption(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "600x400.png"},
		},
	}

	tests := []struct {
		name string
		body string
	}{
		{
			name: "LF",
			body: "<img width=\"600\" height=\"400\" alt=\"600x400.png\" src=\"/attachments/att-1\">\n*サンプル画像です*",
		},
		{
			name: "CRLF",
			body: "<img width=\"600\" height=\"400\" alt=\"600x400.png\" src=\"/attachments/att-1\">\r\n*サンプル画像です*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := RenderHTML(context.Background(), tt.body, "topic1", "space-1", "my-space", resolver, finder)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(got, `data-attachment-id="att-1"`) {
				t.Errorf("result should contain attachment, got: %s", got)
			}
			if !strings.Contains(got, "<em>サンプル画像です</em>") {
				t.Errorf("emphasis should be converted to <em>, got: %s", got)
			}
			if !strings.Contains(got, "<br/>") {
				t.Errorf("result should contain <br/> between image and caption, got: %s", got)
			}
		})
	}
}

func TestRenderHTML_ImageWithCaption(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"},
		},
	}

	body := "![写真](/attachments/att-1)\n*キャプションテキスト*"
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 画像リンクが変換されていること
	if !strings.Contains(got, `data-attachment-id="att-1"`) {
		t.Errorf("result should contain attachment, got: %s", got)
	}
	// 画像リンクとキャプションが同じ<p>でラップされていること
	if !strings.Contains(got, "<em>キャプションテキスト</em></p>") {
		t.Errorf("caption should be wrapped with image in <p>, got: %s", got)
	}
	// <br>が画像とキャプションの間にあること
	if !strings.Contains(got, "<br/>") {
		t.Errorf("result should contain <br/> between image and caption, got: %s", got)
	}
}

func TestRenderHTML_ImgWithBackslashInSrc(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"},
		},
	}

	// src属性にバックスラッシュ付きの不正なHTMLが入力された場合、
	// bluemondayが\を%5Cにエンコードするが、エラーにならずスキップされること
	body := `<img src="/attachments/att-1\">`
	got, err := RenderHTML(context.Background(), body, "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 不正なIDは変換されないが、エラーにならないこと
	if strings.Contains(got, `data-attachment-id`) {
		t.Errorf("backslash URL should not be converted to attachment, got: %s", got)
	}
}
