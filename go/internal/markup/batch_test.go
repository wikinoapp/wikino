package markup

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// mockPageLocationResolver はテスト用のPageLocationResolverモック
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

// mockBatchAttachmentFinder はテスト用のBatchAttachmentFinderモック
type mockBatchAttachmentFinder struct {
	attachments []*model.Attachment
	err         error
}

func (m *mockBatchAttachmentFinder) FindByIDsAndSpace(_ context.Context, _ []model.AttachmentID, _ model.SpaceID) ([]*model.Attachment, error) {
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
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestRenderHTML_PlainMarkdown(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{}

	got, err := RenderHTML(context.Background(), "Hello **world**", "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "<strong>world</strong>") {
		t.Errorf("result should contain bold text, got: %s", got)
	}
}

func TestRenderHTML_WithWikilinks(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "ページA", TopicName: "topic1", PageTitle: "ページA"},
				TopicName:  "topic1",
				PageID:     "page-1",
				PageNumber: 1,
				PageTitle:  "ページA",
			},
		},
	}
	finder := &mockBatchAttachmentFinder{}

	got, err := RenderHTML(context.Background(), "リンク: [[ページA]]", "topic1", "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, `<a href="/s/my-space/pages/1"`) {
		t.Errorf("result should contain wikilink, got: %s", got)
	}
	if !strings.Contains(got, "ページA</a>") {
		t.Errorf("result should contain page title in link, got: %s", got)
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
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, `data-attachment-id="att-1"`) {
		t.Errorf("result should contain attachment, got: %s", got)
	}
}

func TestRenderHTML_MixedContent(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "ページA", TopicName: "topic1", PageTitle: "ページA"},
				TopicName:  "topic1",
				PageID:     "page-1",
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
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "<a") {
		t.Errorf("result should contain wikilink, got: %s", got)
	}
	if !strings.Contains(got, `data-attachment-id="att-1"`) {
		t.Errorf("result should contain attachment, got: %s", got)
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
		t.Fatal("expected error but got nil")
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
		t.Fatal("expected error but got nil")
	}
}

func TestRenderHTMLBatch_EmptyInputs(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{}
	finder := &mockBatchAttachmentFinder{}

	got, err := RenderHTMLBatch(context.Background(), nil, "space-1", "my-space", resolver, finder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
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
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if !strings.Contains(got[0], "<strong>world</strong>") {
		t.Errorf("first result should contain bold text, got: %s", got[0])
	}
	if !strings.Contains(got[1], "<em>text</em>") {
		t.Errorf("second result should contain italic text, got: %s", got[1])
	}
}

func TestRenderHTMLBatch_WithWikilinks(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "ページA", TopicName: "topic1", PageTitle: "ページA"},
				TopicName:  "topic1",
				PageID:     "page-1",
				PageNumber: 1,
				PageTitle:  "ページA",
			},
			{
				Key:        WikilinkKey{Raw: "topic2/ページB", TopicName: "topic2", PageTitle: "ページB"},
				TopicName:  "topic2",
				PageID:     "page-2",
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
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}

	// Wikiリンクが<a>タグに変換されていること
	if !strings.Contains(got[0], "<a") {
		t.Errorf("first result should contain link, got: %s", got[0])
	}
	if !strings.Contains(got[0], "ページA") {
		t.Errorf("first result should contain page title, got: %s", got[0])
	}
	if !strings.Contains(got[1], "<a") {
		t.Errorf("second result should contain link, got: %s", got[1])
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
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}

	// 画像添付ファイルが変換されていること
	if !strings.Contains(got[0], `data-attachment-id="att-1"`) {
		t.Errorf("first result should contain attachment att-1, got: %s", got[0])
	}
	// PDF添付ファイルが変換されていること
	if !strings.Contains(got[1], `data-attachment-id="att-2"`) {
		t.Errorf("second result should contain attachment att-2, got: %s", got[1])
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
		t.Fatal("expected error but got nil")
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
		t.Fatal("expected error but got nil")
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
		t.Fatalf("expected 2 unique keys, got %d", len(got))
	}

	if got[0].TopicName != "topic1" || got[0].PageTitle != "ページA" {
		t.Errorf("first key should be topic1/ページA, got %s/%s", got[0].TopicName, got[0].PageTitle)
	}
	if got[1].TopicName != "topic2" || got[1].PageTitle != "ページB" {
		t.Errorf("second key should be topic2/ページB, got %s/%s", got[1].TopicName, got[1].PageTitle)
	}
}

func TestDeduplicateWikilinkKeys_Empty(t *testing.T) {
	t.Parallel()

	got := deduplicateWikilinkKeys(nil)
	if len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestCollectAllAttachmentIDs(t *testing.T) {
	t.Parallel()

	htmls := []string{
		`<img src="/attachments/att-1"><img src="/attachments/att-2">`,
		`<img src="/attachments/att-2"><img src="/attachments/att-3">`,
		`<img src="/attachments/att-1">`,
	}

	got := collectAllAttachmentIDs(htmls)
	if len(got) != 3 {
		t.Fatalf("expected 3 unique IDs, got %d: %v", len(got), got)
	}

	// 重複が除去され、出現順に並んでいること
	expected := []string{"att-1", "att-2", "att-3"}
	for i, want := range expected {
		if got[i] != want {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want)
		}
	}
}

func TestCollectAllAttachmentIDs_Empty(t *testing.T) {
	t.Parallel()

	got := collectAllAttachmentIDs([]string{"<p>テキスト</p>", "<p>テキスト2</p>"})
	if len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
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
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected attachment, got nil")
	}
	if got.Filename != "photo.jpg" {
		t.Errorf("expected photo.jpg, got %s", got.Filename)
	}

	// 存在しないIDの検索
	got, err = finder.FindByIDAndSpace(context.Background(), "nonexistent", "space-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent ID, got %v", got)
	}
}

func TestMapAttachmentFinder_Empty(t *testing.T) {
	t.Parallel()

	finder := newMapAttachmentFinder(nil)

	got, err := finder.FindByIDAndSpace(context.Background(), "att-1", "space-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestRenderHTMLBatch_MixedContent(t *testing.T) {
	t.Parallel()

	resolver := &mockPageLocationResolver{
		locations: []PageLocation{
			{
				Key:        WikilinkKey{Raw: "ページA", TopicName: "topic1", PageTitle: "ページA"},
				TopicName:  "topic1",
				PageID:     "page-1",
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
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}

	// Wikiリンクと添付ファイルの両方が変換されていること
	if !strings.Contains(got[0], "<a") {
		t.Errorf("first result should contain wikilink, got: %s", got[0])
	}
	if !strings.Contains(got[0], `data-attachment-id="att-1"`) {
		t.Errorf("first result should contain attachment, got: %s", got[0])
	}

	// 普通のテキストはそのまま
	if !strings.Contains(got[1], "普通のテキスト") {
		t.Errorf("second result should contain plain text, got: %s", got[1])
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
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}

	// 両方のテキストで同じ添付ファイルが変換されていること
	for i, html := range got {
		if !strings.Contains(html, `data-attachment-id="att-shared"`) {
			t.Errorf("result[%d] should contain shared attachment, got: %s", i, html)
		}
	}
}
