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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// タスクリストのチェックボックスが含まれること
	if !strings.Contains(got, `type="checkbox"`) {
		t.Errorf("結果にタスクリストのチェックボックスが含まれていない: %s", got)
	}
	// Wikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("結果にWikiリンクが含まれていない: %s", got)
	}
	if !strings.Contains(got, "設計メモ</a>") {
		t.Errorf("結果のリンクにページタイトルが含まれていない: %s", got)
	}
	// チェック済みタスクの存在
	if !strings.Contains(got, `checked`) {
		t.Errorf("結果にチェック済みのチェックボックスが含まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 同一トピックのWikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/10"`) {
		t.Errorf("結果に同じトピックへのWikiリンクが含まれていない: %s", got)
	}
	// クロストピックのWikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/20"`) {
		t.Errorf("結果に別トピックへのWikiリンクが含まれていない: %s", got)
	}
	// 画像添付ファイルが変換されていること
	if !strings.Contains(got, `data-attachment-id="att-img"`) {
		t.Errorf("結果に画像の添付ファイルが含まれていない: %s", got)
	}
	if !strings.Contains(got, `data-attachment-type="image"`) {
		t.Errorf("結果に画像のtype属性が含まれていない: %s", got)
	}
	// PDF添付ファイルがdata属性付きリンクに変換されていること
	if !strings.Contains(got, `data-attachment-id="att-doc"`) {
		t.Errorf("結果にPDFの添付ファイルが含まれていない: %s", got)
	}
	// Markdownリンク記法のテキスト「仕様書」が保持されていること
	if !strings.Contains(got, "仕様書</a>") {
		t.Errorf("結果に元のリンクテキストが保持されていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 通常テキスト内のWikiリンクは変換される
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("結果に通常のテキスト中のWikiリンクが含まれていない: %s", got)
	}
	// コードブロック内のWikiリンクは変換されない ([[ページA]]がそのまま残る)
	if !strings.Contains(got, "<code>") {
		t.Errorf("結果にコードブロックが含まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// リンクタグに変換されないこと
	if strings.Contains(got, `<a href="/s/my-space/pages/`) {
		t.Errorf("存在しないページがリンクになっている: %s", got)
	}
	// 元のテキストが残ること
	if !strings.Contains(got, "存在しないページ") {
		t.Errorf("ページタイトルのテキストが残っていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// data-attachment-id属性が付与されないこと
	if strings.Contains(got, `data-attachment-id`) {
		t.Errorf("存在しない添付ファイルが変換されている: %s", got)
	}
	// img要素自体は残ること (サニタイズされた状態で)
	if !strings.Contains(got, "img") {
		t.Errorf("img要素が残っていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 括弧を含むタイトルが正しくリンクされること
	if !strings.Contains(got, `<a href="/s/my-space/pages/5"`) {
		t.Errorf("結果に括弧を含むタイトルへのリンクが含まれていない: %s", got)
	}
	// 矢印を含むタイトルが正しくリンクされること
	if !strings.Contains(got, `<a href="/s/my-space/pages/6"`) {
		t.Errorf("結果に矢印を含むタイトルへのリンクが含まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// JPG画像: インライン画像として変換
	if !strings.Contains(got, `data-attachment-id="att-jpg"`) {
		t.Errorf("結果にJPGの添付ファイルが含まれていない: %s", got)
	}
	// PNG画像: インライン画像として変換
	if !strings.Contains(got, `data-attachment-id="att-png"`) {
		t.Errorf("結果にPNGの添付ファイルが含まれていない: %s", got)
	}
	// PDF: ダウンロードリンクとして変換 (ファイル名表示)
	if !strings.Contains(got, `data-attachment-id="att-pdf"`) {
		t.Errorf("結果にPDFの添付ファイルが含まれていない: %s", got)
	}
	if !strings.Contains(got, "document.pdf") {
		t.Errorf("結果にPDFのファイル名が表示されていない: %s", got)
	}
	// MP4動画: video要素として変換
	if !strings.Contains(got, `data-attachment-id="att-mp4"`) {
		t.Errorf("結果にMP4の添付ファイルが含まれていない: %s", got)
	}
	if !strings.Contains(got, "<video") {
		t.Errorf("結果にMP4のvideo要素が含まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 見出しが変換されていること
	if !strings.Contains(got, "<h1") {
		t.Errorf("結果にh1の見出しが含まれていない: %s", got)
	}
	if !strings.Contains(got, "<h2") {
		t.Errorf("結果にh2の見出しが含まれていない: %s", got)
	}
	// 太字が変換されていること
	if !strings.Contains(got, "<strong>重要な</strong>") {
		t.Errorf("結果に太字が含まれていない: %s", got)
	}
	// Wikiリンクが変換されていること (リスト内、タスクリスト内の両方)
	if count := strings.Count(got, `href="/s/my-space/pages/1"`); count < 2 {
		t.Errorf("結果のページ1へのリンクの件数 = %d、期待値 = 2以上: %s", count, got)
	}
	if !strings.Contains(got, `<a href="/s/my-space/pages/2"`) {
		t.Errorf("結果に別トピックへのWikiリンクが含まれていない: %s", got)
	}
	// 画像添付ファイルが変換されていること
	if !strings.Contains(got, `data-attachment-id="att-arch"`) {
		t.Errorf("結果にアーキテクチャの画像が含まれていない: %s", got)
	}
	// Excel添付ファイルがリンクに変換されていること
	if !strings.Contains(got, `data-attachment-id="att-spec"`) {
		t.Errorf("結果に仕様書の添付ファイルが含まれていない: %s", got)
	}
	// タスクリストが含まれること
	if !strings.Contains(got, `type="checkbox"`) {
		t.Errorf("結果にタスクリストのチェックボックスが含まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 画像リンクが存在すること
	if !strings.Contains(got, `class="wikino-attachment-image-link"`) {
		t.Errorf("結果に画像リンクのクラスが含まれていない: %s", got)
	}
	// スタンドアロン画像が<p>でラップされていること
	if !strings.Contains(got, `<p><a`) {
		t.Errorf("単独の画像が<p>で囲まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 両方のWikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("結果に1つ目のWikiリンクが含まれていない: %s", got)
	}
	if !strings.Contains(got, `<a href="/s/my-space/pages/2"`) {
		t.Errorf("結果に2つ目のWikiリンクが含まれていない: %s", got)
	}
	if !strings.Contains(got, "ページA</a>") {
		t.Errorf("結果に1つ目のページタイトルが含まれていない: %s", got)
	}
	if !strings.Contains(got, "ページB</a>") {
		t.Errorf("結果に2つ目のページタイトルが含まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// テーブルが生成されていること
	if !strings.Contains(got, "<table") {
		t.Errorf("結果に表が含まれていない: %s", got)
	}
	// テーブル内のWikiリンクが変換されていること
	if !strings.Contains(got, `<a href="/s/my-space/pages/3"`) {
		t.Errorf("結果に表の中のWikiリンクが含まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 存在するページはリンクに変換される
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("既存のページがリンクになっていない: %s", got)
	}
	// 存在しないページはリンクに変換されない
	linkCount := strings.Count(got, `<a href="/s/my-space/pages/`)
	if linkCount != 1 {
		t.Errorf("Wikiリンクの件数 = %d、期待値 = 1: %s", linkCount, got)
	}
	// 存在しないページのテキストは残る
	if !strings.Contains(got, "存在しないページ") {
		t.Errorf("存在しないページのタイトルがテキストとして残っていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 添付ファイルが変換されていること
	if !strings.Contains(got, `data-attachment-id="att-1"`) {
		t.Errorf("結果に添付ファイルが含まれていない: %s", got)
	}
	// 前後のテキストが保持されていること
	if !strings.Contains(got, "テキストの中に") {
		t.Errorf("前後のテキストが保持されていない: %s", got)
	}
	if !strings.Contains(got, "が含まれています") {
		t.Errorf("前後のテキストが保持されていない: %s", got)
	}
}

func TestRenderHTML_HTMLImgWithCaption(t *testing.T) {
	t.Parallel()

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

			resolver := &mockPageLocationResolver{}
			finder := &mockBatchAttachmentFinder{
				attachments: []*model.Attachment{
					{ID: "att-1", SpaceID: "space-1", Filename: "600x400.png"},
				},
			}

			got, err := RenderHTML(context.Background(), tt.body, "topic1", "space-1", "my-space", resolver, finder)
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			if !strings.Contains(got, `data-attachment-id="att-1"`) {
				t.Errorf("結果に添付ファイルが含まれていない: %s", got)
			}
			if !strings.Contains(got, "<em>サンプル画像です</em>") {
				t.Errorf("強調が<em>に変換されていない: %s", got)
			}
			if !strings.Contains(got, "<br/>") {
				t.Errorf("結果の画像とキャプションの間に<br/>が含まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 画像リンクが変換されていること
	if !strings.Contains(got, `data-attachment-id="att-1"`) {
		t.Errorf("結果に添付ファイルが含まれていない: %s", got)
	}
	// 画像リンクとキャプションが同じ<p>でラップされていること
	if !strings.Contains(got, "<em>キャプションテキスト</em></p>") {
		t.Errorf("キャプションが画像と一緒に<p>で囲まれていない: %s", got)
	}
	// <br>が画像とキャプションの間にあること
	if !strings.Contains(got, "<br/>") {
		t.Errorf("結果の画像とキャプションの間に<br/>が含まれていない: %s", got)
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 不正なIDは変換されないが、エラーにならないこと
	if strings.Contains(got, `data-attachment-id`) {
		t.Errorf("バックスラッシュを含むURLが添付ファイルに変換されている: %s", got)
	}
}
