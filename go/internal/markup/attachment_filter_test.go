package markup

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// mockAttachmentFinderはテスト用のAttachmentFinderモック
type mockAttachmentFinder struct {
	attachments map[string]*model.Attachment
	err         error
}

func (m *mockAttachmentFinder) FindByIDAndSpace(_ context.Context, id model.AttachmentID, _ model.SpaceID) (*model.Attachment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.attachments[string(id)], nil
}

func newMockFinder(attachments ...*model.Attachment) *mockAttachmentFinder {
	m := &mockAttachmentFinder{attachments: make(map[string]*model.Attachment)}
	for _, a := range attachments {
		m.attachments[string(a.ID)] = a
	}
	return m
}

func TestFilterAttachments_InlineImage(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-img-1",
		SpaceID:  "space-1",
		Filename: "photo.jpg",
	})

	input := `<p><img src="/attachments/att-img-1"></p>`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	checks := []string{
		`data-attachment-id="att-img-1"`,
		`data-attachment-type="image"`,
		`class="wikino-attachment-image"`,
		`class="wikino-attachment-image-link"`,
		`alt="photo.jpg"`,
		`src=""`,
		`target="_blank"`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("結果に%qが含まれていない\n実測値: %s", check, got)
		}
	}

	if strings.Contains(got, `/attachments/att-img-1`) {
		t.Errorf("結果に元の添付ファイルのURLが含まれている\n実測値: %s", got)
	}
}

func TestFilterAttachments_InlineImageWithWidthAndAlt(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-img-2",
		SpaceID:  "space-1",
		Filename: "banner.png",
	})

	input := `<img src="/attachments/att-img-2" width="300" alt="カスタムAlt">`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	checks := []string{
		`width="300"`,
		`alt="カスタムAlt"`,
		`data-attachment-id="att-img-2"`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("結果に%qが含まれていない\n実測値: %s", check, got)
		}
	}
}

func TestFilterAttachments_InlineImageFormats(t *testing.T) {
	t.Parallel()

	formats := []struct {
		ext      string
		filename string
	}{
		{"jpg", "photo.jpg"},
		{"jpeg", "photo.jpeg"},
		{"png", "image.png"},
		{"gif", "anim.gif"},
		{"svg", "icon.svg"},
		{"webp", "modern.webp"},
		{"JPG", "UPPER.JPG"},
	}

	for _, f := range formats {
		t.Run(f.ext, func(t *testing.T) {
			t.Parallel()

			finder := newMockFinder(&model.Attachment{
				ID:       model.AttachmentID("att-" + f.ext),
				SpaceID:  "space-1",
				Filename: f.filename,
			})

			input := `<img src="/attachments/att-` + f.ext + `">`
			got, err := FilterAttachments(context.Background(), input, "space-1", finder)
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			if !strings.Contains(got, `data-attachment-type="image"`) {
				t.Errorf("%sがインライン画像として描画されていない\n実測値: %s", f.ext, got)
			}
		})
	}
}

func TestFilterAttachments_DownloadLink(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-pdf-1",
		SpaceID:  "space-1",
		Filename: "document.pdf",
	})

	input := `<img src="/attachments/att-pdf-1">`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	checks := []string{
		`data-attachment-id="att-pdf-1"`,
		`data-attachment-link="true"`,
		`document.pdf`,
		`<svg`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("結果に%qが含まれていない\n実測値: %s", check, got)
		}
	}

	if strings.Contains(got, `data-attachment-type="image"`) {
		t.Errorf("PDFがインライン画像として描画されている\n実測値: %s", got)
	}
}

func TestFilterAttachments_InlineVideo(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-vid-1",
		SpaceID:  "space-1",
		Filename: "movie.mp4",
	})

	input := `<a href="/attachments/att-vid-1">動画リンク</a>`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	checks := []string{
		`<video`,
		`data-attachment-id="att-vid-1"`,
		`data-attachment-type="video"`,
		`class="wikino-attachment-video"`,
		`controls`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("結果に%qが含まれていない\n実測値: %s", check, got)
		}
	}
}

func TestFilterAttachments_InlineVideoFormats(t *testing.T) {
	t.Parallel()

	formats := []string{"mp4", "webm", "ogg", "mov"}

	for _, ext := range formats {
		t.Run(ext, func(t *testing.T) {
			t.Parallel()

			finder := newMockFinder(&model.Attachment{
				ID:       model.AttachmentID("att-" + ext),
				SpaceID:  "space-1",
				Filename: "video." + ext,
			})

			input := `<a href="/attachments/att-` + ext + `">link</a>`
			got, err := FilterAttachments(context.Background(), input, "space-1", finder)
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			if !strings.Contains(got, `data-attachment-type="video"`) {
				t.Errorf("%sがインライン動画として描画されていない\n実測値: %s", ext, got)
			}
		})
	}
}

func TestFilterAttachments_AnchorWithDataAttrs(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-doc-1",
		SpaceID:  "space-1",
		Filename: "readme.txt",
	})

	input := `<a href="/attachments/att-doc-1">テキストファイル</a>`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	checks := []string{
		`data-attachment-id="att-doc-1"`,
		`data-attachment-link="true"`,
		`テキストファイル`,
		`target="_blank"`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("結果に%qが含まれていない\n実測値: %s", check, got)
		}
	}
}

func TestFilterAttachments_NonAttachmentElementsPreserved(t *testing.T) {
	t.Parallel()

	finder := newMockFinder()

	input := `<p><img src="https://example.com/photo.jpg" alt="external"></p><p><a href="https://example.com">外部リンク</a></p>`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if got != input {
		t.Errorf("添付ファイル以外の要素が保持されていない\n実測値: %s\n期待値: %s", got, input)
	}
}

func TestFilterAttachments_NonExistentAttachmentSkipped(t *testing.T) {
	t.Parallel()

	finder := newMockFinder()

	input := `<img src="/attachments/nonexistent">`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if got != input {
		t.Errorf("存在しない添付ファイルが読み飛ばされていない\n実測値: %s\n期待値: %s", got, input)
	}
}

func TestFilterAttachments_EmptyHTML(t *testing.T) {
	t.Parallel()

	finder := newMockFinder()

	got, err := FilterAttachments(context.Background(), "", "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if got != "" {
		t.Errorf("入力が空なのに空文字列を返さなかった: %q", got)
	}
}

func TestFilterAttachments_MixedContent(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(
		&model.Attachment{ID: "att-1", SpaceID: "space-1", Filename: "photo.png"},
		&model.Attachment{ID: "att-2", SpaceID: "space-1", Filename: "video.mp4"},
	)

	input := `<p>テキスト<img src="/attachments/att-1">テキスト2</p><p><a href="/attachments/att-2">動画</a></p><p><a href="https://example.com">外部</a></p>`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	checks := []string{
		`テキスト`,
		`テキスト2`,
		`data-attachment-type="image"`,
		`data-attachment-type="video"`,
		`https://example.com`,
		`外部`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("結果に%qが含まれていない\n実測値: %s", check, got)
		}
	}
}

func TestFilterAttachments_FinderError(t *testing.T) {
	t.Parallel()

	finder := &mockAttachmentFinder{
		attachments: make(map[string]*model.Attachment),
		err:         errors.New("DB接続エラー"),
	}

	input := `<img src="/attachments/att-1">`
	_, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}
}

func TestFilterAttachments_SelfClosingImgTag(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-1",
		SpaceID:  "space-1",
		Filename: "photo.jpg",
	})

	input := `<img src="/attachments/att-1" />`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if !strings.Contains(got, `data-attachment-id="att-1"`) {
		t.Errorf("自己終了したimgタグが処理されていない\n実測値: %s", got)
	}
}

func TestWrapStandaloneImageLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "スタンドアロン画像リンクを<p>で囲む",
			input: `<a class="wikino-attachment-image-link" href="#" target="_blank"><img src="" alt="photo.jpg"></a>`,
			want:  `<p><a class="wikino-attachment-image-link" href="#" target="_blank"><img src="" alt="photo.jpg"/></a></p>`,
		},
		{
			name:  "<p>内の画像リンクはそのまま",
			input: `<p><a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"></a></p>`,
			want:  `<p><a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"></a></p>`,
		},
		{
			name:  "直後に<br>が続く場合はラッピングしない",
			input: `<a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"></a><br>`,
			want:  `<a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"></a><br>`,
		},
		{
			name:  "直後に<em>が続く場合はラッピングしない",
			input: `<a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"></a><em>caption</em>`,
			want:  `<a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"></a><em>caption</em>`,
		},
		{
			name:  "空白テキストノードを挟んで<em>が続く場合もラッピングしない",
			input: "<a class=\"wikino-attachment-image-link\" href=\"#\"><img src=\"\" alt=\"photo.jpg\"></a>\n<em>caption</em>",
			want:  "<a class=\"wikino-attachment-image-link\" href=\"#\"><img src=\"\" alt=\"photo.jpg\"></a>\n<em>caption</em>",
		},
		{
			name:  "画像リンク以外の要素は変更しない",
			input: `<p>テキスト</p><div>コンテンツ</div>`,
			want:  `<p>テキスト</p><div>コンテンツ</div>`,
		},
		{
			name:  "空文字列は空文字列を返す",
			input: "",
			want:  "",
		},
		{
			name:  "直後に<br>と<em>が続く場合はキャプション付きで<p>で囲む",
			input: `<a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"></a><br><em>キャプション</em>`,
			want:  `<p><a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"/></a><br/><em>キャプション</em></p>`,
		},
		{
			name:  "直後に<br>と<strong>が続く場合もキャプション付きで<p>で囲む",
			input: `<a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"></a><br><strong>キャプション</strong>`,
			want:  `<p><a class="wikino-attachment-image-link" href="#"><img src="" alt="photo.jpg"/></a><br/><strong>キャプション</strong></p>`,
		},
		{
			name:  "空白テキストノードを挟んで<br>と<em>が続く場合もキャプション付きで<p>で囲む",
			input: "<a class=\"wikino-attachment-image-link\" href=\"#\"><img src=\"\" alt=\"photo.jpg\"></a>\n<br>\n<em>キャプション</em>",
			want:  "<p><a class=\"wikino-attachment-image-link\" href=\"#\"><img src=\"\" alt=\"photo.jpg\"/></a>\n<br/>\n<em>キャプション</em></p>",
		},
		{
			name:  "複数のスタンドアロン画像リンクをそれぞれ<p>で囲む",
			input: `<a class="wikino-attachment-image-link" href="#"><img src="" alt="1.jpg"></a><a class="wikino-attachment-image-link" href="#"><img src="" alt="2.jpg"></a>`,
			want:  `<p><a class="wikino-attachment-image-link" href="#"><img src="" alt="1.jpg"/></a></p><p><a class="wikino-attachment-image-link" href="#"><img src="" alt="2.jpg"/></a></p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := WrapStandaloneImageLinks(tt.input)
			if got != tt.want {
				t.Errorf("WrapStandaloneImageLinks()の結果が一致しない\n実測値: %s\n期待値: %s", got, tt.want)
			}
		})
	}
}

// spaceAwareMockFinderはスペースIDも検証するテスト用のAttachmentFinderモック。
// 実際のリポジトリと同様に、スペースが一致しない添付ファイルはnilを返す。
type spaceAwareMockFinder struct {
	attachments map[string]*model.Attachment
}

func (m *spaceAwareMockFinder) FindByIDAndSpace(_ context.Context, id model.AttachmentID, spaceID model.SpaceID) (*model.Attachment, error) {
	att := m.attachments[string(id)]
	if att == nil {
		return nil, nil
	}
	if att.SpaceID != spaceID {
		return nil, nil
	}
	return att, nil
}

func TestFilterAttachments_CrossSpaceAccessBlocked(t *testing.T) {
	t.Parallel()

	finder := &spaceAwareMockFinder{
		attachments: map[string]*model.Attachment{
			"att-cross-1": {ID: "att-cross-1", SpaceID: "space-2", Filename: "photo.jpg"},
		},
	}

	input := `<img src="/attachments/att-cross-1">`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if strings.Contains(got, `data-attachment-id`) {
		t.Errorf("別スペースの添付ファイルが変換されている\n実測値: %s", got)
	}
	if !strings.Contains(got, `/attachments/att-cross-1`) {
		t.Errorf("元のURLが保持されていない\n実測値: %s", got)
	}
}

func TestFilterAttachments_XSSFilenameEscaped(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-xss-1",
		SpaceID:  "space-1",
		Filename: `<script>alert('XSS')</script>.pdf`,
	})

	input := `<img src="/attachments/att-xss-1">`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if strings.Contains(got, "<script>") {
		t.Errorf("<script>タグを含むファイル名がエスケープされていない\n実測値: %s", got)
	}
	if !strings.Contains(got, `data-attachment-id="att-xss-1"`) {
		t.Errorf("添付ファイルが変換されていない\n実測値: %s", got)
	}
}

func TestFilterAttachments_MultipleHTMLImgTags(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(
		&model.Attachment{ID: "att-multi-1", SpaceID: "space-1", Filename: "image1.jpg"},
		&model.Attachment{ID: "att-multi-2", SpaceID: "space-1", Filename: "image2.png"},
		&model.Attachment{ID: "att-multi-3", SpaceID: "space-1", Filename: "document.pdf"},
	)

	input := `<p>First image:</p><img src="/attachments/att-multi-1" alt="image1"><p>Second image:</p><img src="/attachments/att-multi-2" alt="image2"><p>PDF as image:</p><img src="/attachments/att-multi-3" alt="pdf">`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 画像1と画像2はインライン画像として変換されること
	if strings.Count(got, `data-attachment-type="image"`) != 2 {
		t.Errorf("インライン画像が2つになっていない\n実測値: %s", got)
	}
	if !strings.Contains(got, `data-attachment-id="att-multi-1"`) {
		t.Errorf("添付ファイル1が変換されていない\n実測値: %s", got)
	}
	if !strings.Contains(got, `data-attachment-id="att-multi-2"`) {
		t.Errorf("添付ファイル2が変換されていない\n実測値: %s", got)
	}

	// PDFはダウンロードリンクに変換されること
	if !strings.Contains(got, `data-attachment-id="att-multi-3"`) {
		t.Errorf("添付ファイル3が変換されていない\n実測値: %s", got)
	}
	if !strings.Contains(got, "document.pdf") {
		t.Errorf("ダウンロードリンクにPDFのファイル名が含まれていない\n実測値: %s", got)
	}
}

func TestFilterAttachments_ImageFollowedByEmphasis(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-em-1",
		SpaceID:  "space-1",
		Filename: "photo.png",
	})

	// FilterAttachments後の画像リンク直後に<em>が続くケース
	input := `<p><a class="wikino-attachment-image-link" href="/attachments/att-em-1"><img src="/attachments/att-em-1" alt="photo.png"></a><br><em>サンプル画像です</em></p>`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	// 画像が変換されていること
	if !strings.Contains(got, `data-attachment-type="image"`) {
		t.Errorf("画像が変換されていない\n実測値: %s", got)
	}

	// <em>要素が保持されていること
	if !strings.Contains(got, "<em>サンプル画像です</em>") {
		t.Errorf("強調要素が保持されていない\n実測値: %s", got)
	}
}

func TestWrapStandaloneImageLinks_IntegrationWithFilterAttachments(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-wrap-1",
		SpaceID:  "space-1",
		Filename: "600x400.png",
	})

	// FilterAttachmentsの出力をWrapStandaloneImageLinksに渡す統合テスト
	input := `<p>Test</p><img src="/attachments/att-wrap-1" width="600" alt="600x400.png"><p>Test</p>`
	filtered, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	got := WrapStandaloneImageLinks(filtered)

	// 画像リンクが<p>要素で囲まれていること
	if !strings.Contains(got, "<p><a") {
		t.Errorf("単独の画像リンクが<p>で囲まれていない\n実測値: %s", got)
	}
	if !strings.Contains(got, `class="wikino-attachment-image-link"`) {
		t.Errorf("画像リンクのクラスが含まれていない\n実測値: %s", got)
	}
	if !strings.Contains(got, `data-attachment-id="att-wrap-1"`) {
		t.Errorf("添付ファイルのIDが含まれていない\n実測値: %s", got)
	}
	if !strings.Contains(got, `width="600"`) {
		t.Errorf("width属性が保持されていない\n実測値: %s", got)
	}
}

func TestExtractAttachmentID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "正常なパス",
			url:  "/attachments/abc123",
			want: "abc123",
		},
		{
			name: "ULIDのパス",
			url:  "/attachments/01HWXYZ123456789ABCDEF",
			want: "01HWXYZ123456789ABCDEF",
		},
		{
			name: "外部URL",
			url:  "https://example.com/attachments/abc123",
			want: "",
		},
		{
			name: "パスなし",
			url:  "/other/path",
			want: "",
		},
		{
			name: "サブパスあり",
			url:  "/attachments/abc123/extra",
			want: "",
		},
		{
			name: "空文字列",
			url:  "",
			want: "",
		},
		{
			name: "attachmentsのみ",
			url:  "/attachments/",
			want: "",
		},
		{
			name: "パーセントエンコードされたバックスラッシュ",
			url:  "/attachments/att-1%5C",
			want: "",
		},
		{
			name: "バックスラッシュを含むURL",
			url:  `/attachments/att-1\`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := extractAttachmentID(tt.url)
			if got != tt.want {
				t.Errorf("extractAttachmentID(%q) = %q、期待値 = %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestFilterAttachments_PercentEncodedBackslash(t *testing.T) {
	t.Parallel()

	finder := newMockFinder(&model.Attachment{
		ID:       "att-1",
		SpaceID:  "space-1",
		Filename: "photo.png",
	})

	// bluemondayがバックスラッシュを%5Cにエンコードしたケース
	input := `<img src="/attachments/att-1%5C">`
	got, err := FilterAttachments(context.Background(), input, "space-1", finder)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	// バックスラッシュを含む不正なIDは変換されないこと
	if strings.Contains(got, `data-attachment-id`) {
		t.Errorf("パーセントエンコードしたバックスラッシュを含むURLが変換されている\n実測値: %s", got)
	}
}

func TestFileExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "jpg", filename: "photo.jpg", want: "jpg"},
		{name: "PNG (大文字)", filename: "IMAGE.PNG", want: "png"},
		{name: "複数ドット", filename: "archive.tar.gz", want: "gz"},
		{name: "拡張子なし", filename: "README", want: ""},
		{name: "空文字列", filename: "", want: ""},
		{name: "ドットのみ", filename: ".gitignore", want: "gitignore"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := fileExtension(tt.filename)
			if got != tt.want {
				t.Errorf("fileExtension(%q) = %q、期待値 = %q", tt.filename, got, tt.want)
			}
		})
	}
}
