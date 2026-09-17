package markup

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// mockPageLocationResolverはテスト用のPageLocationResolverモック
type mockPageLocationResolver struct {
	locations []PageLocation
	err       error
}

func (m *mockPageLocationResolver) ResolveByKeys(_ context.Context, _ []WikilinkKey, _ model.SpaceID) ([]PageLocation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.locations, nil
}

// mockBatchAttachmentFinderはテスト用のBatchAttachmentFinderモック
type mockBatchAttachmentFinder struct {
	attachments []*model.Attachment
	err         error

	// requestedIDsはバッチが要求したものを記録する。本文がどの添付ファイルを参照している
	// と読まれたかをテストで確かめられるようにするため
	requestedIDs []model.AttachmentID
}

func (m *mockBatchAttachmentFinder) FindByIDsAndSpace(_ context.Context, ids []model.AttachmentID, _ model.SpaceID) ([]*model.Attachment, error) {
	m.requestedIDs = append(m.requestedIDs, ids...)
	if m.err != nil {
		return nil, m.err
	}
	return m.attachments, nil
}

func TestRenderHTML_EmptyBody(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{}

	got, err := RenderHTML(context.Background(), "", "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if got != "" {
		t.Errorf("実測値 = %q、期待値 = 空文字列", got)
	}
}

func TestRenderHTML_PlainMarkdown(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{}

	got, err := RenderHTML(context.Background(), "Hello **world**", "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if !strings.Contains(got, "<strong>world</strong>") {
		t.Errorf("結果に太字が含まれていない: %s", got)
	}
}

func TestRenderHTML_WithWikilinks(t *testing.T) {
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

	got, err := RenderHTML(context.Background(), "リンク: [[ページA]]", "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("結果にWikiリンクが含まれていない: %s", got)
	}
	if !strings.Contains(got, "ページA</a>") {
		t.Errorf("結果のリンクにページタイトルが含まれていない: %s", got)
	}
}

func TestRenderHTML_WithAttachments(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"},
		},
	}

	got, err := RenderHTML(context.Background(), "画像: ![alt](/attachments/att-1)", "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if !strings.Contains(got, `data-attachment-id="att-1"`) {
		t.Errorf("結果に添付ファイルが含まれていない: %s", got)
	}
}

func TestRenderHTML_MixedContent(t *testing.T) {
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
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"},
		},
	}

	got, err := RenderHTML(context.Background(), "リンク: [[ページA]] と画像: ![alt](/attachments/att-1)", "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if !strings.Contains(got, "<a") {
		t.Errorf("結果にWikiリンクが含まれていない: %s", got)
	}
	if !strings.Contains(got, `data-attachment-id="att-1"`) {
		t.Errorf("結果に添付ファイルが含まれていない: %s", got)
	}
}

func TestRenderHTML_ResolverError(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		err: errors.New("DB接続エラー"),
	}
	finder := &mockBatchAttachmentFinder{}

	_, err := RenderHTML(context.Background(), "リンク: [[ページA]]", "topic1", "space-1", "my-space", resolver, finder)
	if err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}
}

func TestRenderHTML_BatchFinderError(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		err: errors.New("DB接続エラー"),
	}

	_, err := RenderHTML(context.Background(), "画像: ![alt](/attachments/att-1)", "topic1", "space-1", "my-space", resolver, finder)
	if err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}
}

func TestRenderHTMLBatch_EmptyInputs(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{}

	got, err := RenderHTMLBatch(context.Background(), nil, "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if got != nil {
		t.Errorf("実測値 = %v、期待値 = nil", got)
	}
}

func TestRenderHTMLBatch_PlainMarkdown(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{}

	inputs := []BatchRenderInput{
		{Body: "Hello **world**", CurrentTopicName: "topic1"},
		{Body: "Another *text*", CurrentTopicName: "topic2"},
	}

	got, err := RenderHTMLBatch(context.Background(), inputs, "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("結果の件数 = %d、期待値 = 2", len(got))
	}
	if !strings.Contains(got[0], "<strong>world</strong>") {
		t.Errorf("1つ目の結果に太字が含まれていない: %s", got[0])
	}
	if !strings.Contains(got[1], "<em>text</em>") {
		t.Errorf("2つ目の結果に斜体が含まれていない: %s", got[1])
	}
}

func TestRenderHTMLBatch_WithWikilinks(t *testing.T) {
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

	inputs := []BatchRenderInput{
		{Body: "リンク: [[ページA]]", CurrentTopicName: "topic1"},
		{Body: "リンク: [[topic2/ページB]]", CurrentTopicName: "topic1"},
	}

	got, err := RenderHTMLBatch(context.Background(), inputs, "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("結果の件数 = %d、期待値 = 2", len(got))
	}

	// Wikiリンクが<a>タグに変換されていること
	if !strings.Contains(got[0], "<a") {
		t.Errorf("1つ目の結果にリンクが含まれていない: %s", got[0])
	}
	if !strings.Contains(got[0], "ページA") {
		t.Errorf("1つ目の結果にページタイトルが含まれていない: %s", got[0])
	}
	if !strings.Contains(got[1], "<a") {
		t.Errorf("2つ目の結果にリンクが含まれていない: %s", got[1])
	}
}

func TestRenderHTMLBatch_WithAttachments(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"},
			{ID: "att-2", SpaceID: "space-1", Filename: "doc.pdf"},
		},
	}

	inputs := []BatchRenderInput{
		{Body: "画像: ![alt](/attachments/att-1)", CurrentTopicName: "topic1"},
		{Body: "文書: ![alt](/attachments/att-2)", CurrentTopicName: "topic2"},
	}

	got, err := RenderHTMLBatch(context.Background(), inputs, "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("結果の件数 = %d、期待値 = 2", len(got))
	}

	// 画像添付ファイルが変換されていること
	if !strings.Contains(got[0], `data-attachment-id="att-1"`) {
		t.Errorf("1つ目の結果に添付ファイルatt-1が含まれていない: %s", got[0])
	}
	// PDF添付ファイルが変換されていること
	if !strings.Contains(got[1], `data-attachment-id="att-2"`) {
		t.Errorf("2つ目の結果に添付ファイルatt-2が含まれていない: %s", got[1])
	}
}

func TestRenderHTMLBatch_ResolverError(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		err: errors.New("DB接続エラー"),
	}
	finder := &mockBatchAttachmentFinder{}

	inputs := []BatchRenderInput{
		{Body: "リンク: [[ページA]]", CurrentTopicName: "topic1"},
	}

	_, err := RenderHTMLBatch(context.Background(), inputs, "space-1", "my-space", resolver, finder)
	if err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}
}

func TestRenderHTMLBatch_BatchFinderError(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		err: errors.New("DB接続エラー"),
	}

	inputs := []BatchRenderInput{
		{Body: "画像: ![alt](/attachments/att-1)", CurrentTopicName: "topic1"},
	}

	_, err := RenderHTMLBatch(context.Background(), inputs, "space-1", "my-space", resolver, finder)
	if err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}
}

func TestDeduplicateWikilinkKeys(t *testing.T) {
	t.Parallel()

	keys := []WikilinkKey{
		{Raw: "topic1/ページA", TopicName: "topic1", PageTitle: "ページA"},
		{Raw: "topic1/ページA", TopicName: "topic1", PageTitle: "ページA"},
		{Raw: "topic2/ページB", TopicName: "topic2", PageTitle: "ページB"},
		{Raw: "topic1/ページA", TopicName: "topic1", PageTitle: "ページA"},
		{Raw: "topic2/ページB", TopicName: "topic2", PageTitle: "ページB"},
	}

	got := deduplicateWikilinkKeys(keys)
	if len(got) != 2 {
		t.Fatalf("一意なキーの件数 = %d、期待値 = 2", len(got))
	}

	if got[0].TopicName != "topic1" || got[0].PageTitle != "ページA" {
		t.Errorf("最初のキー = %s/%s、期待値 = topic1/ページA", got[0].TopicName, got[0].PageTitle)
	}
	if got[1].TopicName != "topic2" || got[1].PageTitle != "ページB" {
		t.Errorf("2つ目のキー = %s/%s、期待値 = topic2/ページB", got[1].TopicName, got[1].PageTitle)
	}
}

func TestDeduplicateWikilinkKeys_Empty(t *testing.T) {
	t.Parallel()

	got := deduplicateWikilinkKeys(nil)
	if len(got) != 0 {
		t.Errorf("実測値 = %d、期待値 = 0", len(got))
	}
}

func TestCollectAllAttachmentIDs(t *testing.T) {
	t.Parallel()

	perBody := [][]string{
		{"att-1", "att-2"},
		{"att-2", "att-3"},
		{"att-1"},
	}

	got := collectAllAttachmentIDs(perBody)
	if len(got) != 3 {
		t.Fatalf("一意なIDの件数 = %d、期待値 = 3: %v", len(got), got)
	}

	// 重複が除去され、出現順に並んでいること
	expected := []string{"att-1", "att-2", "att-3"}
	for i, want := range expected {
		if got[i] != want {
			t.Errorf("got[%d] = %q、期待値 = %q", i, got[i], want)
		}
	}
}

func TestCollectAllAttachmentIDs_Empty(t *testing.T) {
	t.Parallel()

	got := collectAllAttachmentIDs([][]string{nil, nil})
	if len(got) != 0 {
		t.Errorf("実測値 = %d、期待値 = 0", len(got))
	}
}

func TestMapAttachmentFinder(t *testing.T) {
	t.Parallel()

	attachments := []*model.Attachment{
		{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"},
		{ID: "att-2", SpaceID: "space-1", Filename: "doc.pdf"},
	}

	finder := newMapAttachmentFinder(attachments)

	// 存在するIDの検索
	got, err := finder.FindByIDAndSpace(context.Background(), "att-1", "space-1")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if got == nil {
		t.Fatal("添付ファイルがnil")
	}
	if got.Filename != "photo.jpg" {
		t.Errorf("実測値 = %s、期待値 = photo.jpg", got.Filename)
	}

	// 存在しないIDの検索
	got, err = finder.FindByIDAndSpace(context.Background(), "nonexistent", "space-1")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if got != nil {
		t.Errorf("存在しないIDの結果 = %v、期待値 = nil", got)
	}
}

func TestMapAttachmentFinder_Empty(t *testing.T) {
	t.Parallel()

	finder := newMapAttachmentFinder(nil)

	got, err := finder.FindByIDAndSpace(context.Background(), "att-1", "space-1")
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if got != nil {
		t.Errorf("実測値 = %v、期待値 = nil", got)
	}
}

func TestRenderHTMLBatch_MixedContent(t *testing.T) {
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
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"},
		},
	}

	inputs := []BatchRenderInput{
		{Body: "リンク: [[ページA]] と画像: ![alt](/attachments/att-1)", CurrentTopicName: "topic1"},
		{Body: "普通のテキスト", CurrentTopicName: "topic2"},
	}

	got, err := RenderHTMLBatch(context.Background(), inputs, "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("結果の件数 = %d、期待値 = 2", len(got))
	}

	// Wikiリンクと添付ファイルの両方が変換されていること
	if !strings.Contains(got[0], "<a") {
		t.Errorf("1つ目の結果にWikiリンクが含まれていない: %s", got[0])
	}
	if !strings.Contains(got[0], `data-attachment-id="att-1"`) {
		t.Errorf("1つ目の結果に添付ファイルが含まれていない: %s", got[0])
	}

	// 普通のテキストはそのまま
	if !strings.Contains(got[1], "普通のテキスト") {
		t.Errorf("2つ目の結果にプレーンテキストが含まれていない: %s", got[1])
	}
}

func TestRenderHTMLBatch_SharedAttachmentAcrossInputs(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-shared", SpaceID: "space-1", Filename: "shared.jpg"},
		},
	}

	inputs := []BatchRenderInput{
		{Body: "![alt](/attachments/att-shared)", CurrentTopicName: "topic1"},
		{Body: "![alt](/attachments/att-shared)", CurrentTopicName: "topic2"},
	}

	got, err := RenderHTMLBatch(context.Background(), inputs, "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("結果の件数 = %d、期待値 = 2", len(got))
	}

	// 両方のテキストで同じ添付ファイルが変換されていること
	for i, html := range got {
		if !strings.Contains(html, `data-attachment-id="att-shared"`) {
			t.Errorf("result[%d]に共有の添付ファイルが含まれていない: %s", i, html)
		}
	}
}

func TestRenderHTML_ReferenceDefinitionInsideHiddenContent(t *testing.T) {
	t.Parallel()

	for _, element := range []string{"object", "textarea"} {
		t.Run(element, func(t *testing.T) {
			t.Parallel()

			body := "[visible][r]\n\nx <" + element + ">\n\n[r]: /attachments/att-1\n\n</" + element + ">"
			finder := &mockBatchAttachmentFinder{
				attachments: []*model.Attachment{{ID: "att-1", SpaceID: "space-1", Filename: "file.pdf"}},
			}
			got, err := RenderHTML(context.Background(), body, "T", "space-1", "space", &mockPageLocationResolver{}, finder)
			if err != nil {
				t.Fatalf("RenderHTML()のエラー = %v", err)
			}
			if !strings.Contains(got, `data-attachment-id="att-1"`) || !strings.Contains(got, ">visible</a>") {
				t.Errorf("有効な添付ファイルのリンクが変換されていない: %s", got)
			}
		})
	}
}

func TestRenderHTMLBatch_AttachmentIDsComeFromTheRenderedBodies(t *testing.T) {
	t.Parallel()

	inputs := []BatchRenderInput{
		{Body: `<img src="/attachments/att-1">`, CurrentTopicName: "T"},
		{Body: "><!--\n\n" + `<img src="/attachments/att-2">`, CurrentTopicName: "T"},
		{Body: "`" + `<img src="/attachments/att-3">` + "`", CurrentTopicName: "T"},
	}
	finder := &mockBatchAttachmentFinder{
		attachments: []*model.Attachment{
			{ID: "att-1", SpaceID: "space-1", Filename: "one.png"},
			{ID: "att-2", SpaceID: "space-1", Filename: "two.png"},
			{ID: "att-3", SpaceID: "space-1", Filename: "three.png"},
		},
	}

	got, err := RenderHTMLBatch(context.Background(), inputs, "space-1", "space", &mockPageLocationResolver{}, finder)
	if err != nil {
		t.Fatalf("RenderHTMLBatch()のエラー = %v", err)
	}
	if !strings.Contains(got[0], `data-attachment-id="att-1"`) {
		t.Errorf("有効な添付ファイルが変換されていない: %s", got[0])
	}
	// どちらの本文も読み手に添付ファイルを指すものが出ないため、一括検索へ渡さない。
	if len(finder.requestedIDs) != 1 || finder.requestedIDs[0] != "att-1" {
		t.Errorf("要求されたID = %v、期待値 = att-1のみ", finder.requestedIDs)
	}
}

func TestRenderHTMLBatch_ClosesUnclosedElements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		wantText string
		// wantHTMLは出力に含まれるべきHTML。工程が変換したものも直列化まで残ることを確かめる
		wantHTML          string
		wantExternalLinks int
	}{
		{name: "閉じていないpre", body: "<pre>x", wantText: "x"},
		{name: "入れ子で閉じていないdivとspan", body: "<div><span>z", wantText: "z"},
		{name: "table要素の外にあるtd", body: "<td>y", wantText: "y"},
		{name: "閉じていない表", body: "<table><tr><td>a", wantText: "a"},
		{name: "段落をまたいで閉じていないem", body: "<em>強調\n\n次の段落", wantText: "強調 次の段落"},
		{name: "閉じていないdetails", body: "本文\n\n<details><summary>s</summary>\n\n中身", wantText: "本文 s 中身"},
		{name: "Wikiリンクを含む閉じていないdiv", body: "<div>\n\n[[ページA]]", wantText: "ページA", wantHTML: `<a href="/s/my-space/pages/1"`},
		{name: "添付ファイルを含む閉じていないdiv", body: "<div>\n\n![写真](/attachments/att-1)", wantText: "", wantHTML: `data-attachment-id="att-1"`},
		{name: "リンク内の添付画像", body: `<div><a href="https://example.com"><img src="/attachments/att-1"></a></div>`, wantHTML: `data-attachment-id="att-1"`, wantExternalLinks: 1},
		{name: "リンク内の添付画像とキャプション", body: `<div><a href="https://example.com"><img src="/attachments/att-1"><br><em>説明</em></a></div>`, wantText: "説明", wantHTML: `data-attachment-id="att-1"`, wantExternalLinks: 1},
		{name: "Markdownのリンク付き画像", body: `[![写真](/attachments/att-1)](https://example.com)`, wantHTML: `data-attachment-id="att-1"`, wantExternalLinks: 1},
		{name: "強調を挟むリンク付き画像", body: `[**前![写真](/attachments/att-1)後**末尾](https://example.com)`, wantText: "前後末尾", wantHTML: `data-attachment-id="att-1"`, wantExternalLinks: 1},
		{name: "複数のリンク付き画像", body: `<div><a href="https://example.com">前<img src="/attachments/att-1">中<img src="/attachments/att-1">後</a></div>`, wantText: "前 中 後", wantHTML: `data-attachment-id="att-1"`, wantExternalLinks: 1},
	}

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			finder := &mockBatchAttachmentFinder{
				attachments: []*model.Attachment{{ID: "att-1", SpaceID: "space-1", Filename: "photo.jpg"}},
			}
			inputs := []BatchRenderInput{{Body: tt.body, CurrentTopicName: "topic1"}}
			got, err := RenderHTMLBatch(context.Background(), inputs, "space-1", "my-space", resolver, finder)
			if err != nil {
				t.Fatalf("RenderHTMLBatch()のエラー = %v", err)
			}

			// ツリーを直列化したHTMLは、パースして直列化し直しても変わらない。
			container, err := parseHTMLFragmentWithContainer(got[0])
			if err != nil {
				t.Fatalf("出力のパースのエラー = %v", err)
			}
			reparsed := renderContainerChildren(container)
			if !slices.Equal(htmlTokens(got[0]), htmlTokens(reparsed)) {
				t.Errorf("出力の要素の対応が取れていない: 出力 = %q、パースし直した結果 = %q", got[0], reparsed)
			}
			if text := PlainText(got[0], 0); text != tt.wantText {
				t.Errorf("出力のテキスト = %q、期待値 = %q (出力 = %q)", text, tt.wantText, got[0])
			}
			if count := strings.Count(got[0], `href="https://example.com"`); count != tt.wantExternalLinks {
				t.Errorf("外部リンクの数 = %d、期待値 = %d: %s", count, tt.wantExternalLinks, got[0])
			}
			if !strings.Contains(got[0], tt.wantHTML) {
				t.Errorf("出力に %qが含まれていない: %s", tt.wantHTML, got[0])
			}
		})
	}
}

// htmlTokensはsをトークンの文字列表現の列にする。パーサーは書式要素の属性を並べ替えて
// 保持するため、直列化し直した結果と比べられるよう各トークンの属性はキーの順に並べる。
func htmlTokens(s string) []string {
	var tokens []string
	tokenizer := html.NewTokenizer(strings.NewReader(s))
	for tokenizer.Next() != html.ErrorToken {
		token := tokenizer.Token()
		slices.SortFunc(token.Attr, func(a, b html.Attribute) int {
			return strings.Compare(a.Key, b.Key)
		})
		tokens = append(tokens, token.String())
	}

	return tokens
}
