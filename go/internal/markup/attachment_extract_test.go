package markup

import (
	"testing"
)

func TestExtractAttachmentIDs_HTMLImgTag(t *testing.T) {
	t.Parallel()

	body := `<p><img src="/attachments/abc-123-def" alt="画像"></p>`
	got := ExtractAttachmentIDs(body)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0] != "abc-123-def" {
		t.Errorf("got[0] = %q, want %q", got[0], "abc-123-def")
	}
}

func TestExtractAttachmentIDs_HTMLATag(t *testing.T) {
	t.Parallel()

	body := `<p><a href="/attachments/xyz-456-uvw">ファイル</a></p>`
	got := ExtractAttachmentIDs(body)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0] != "xyz-456-uvw" {
		t.Errorf("got[0] = %q, want %q", got[0], "xyz-456-uvw")
	}
}

func TestExtractAttachmentIDs_MarkdownImage(t *testing.T) {
	t.Parallel()

	body := `![サムネイル画像](/attachments/md-img-001)`
	got := ExtractAttachmentIDs(body)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0] != "md-img-001" {
		t.Errorf("got[0] = %q, want %q", got[0], "md-img-001")
	}
}

func TestExtractAttachmentIDs_MarkdownLink(t *testing.T) {
	t.Parallel()

	body := `[ダウンロード](/attachments/md-link-001)`
	got := ExtractAttachmentIDs(body)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0] != "md-link-001" {
		t.Errorf("got[0] = %q, want %q", got[0], "md-link-001")
	}
}

func TestExtractAttachmentIDs_MarkdownLinkExcludesImage(t *testing.T) {
	t.Parallel()

	body := `![画像](/attachments/img-only)`
	got := ExtractAttachmentIDs(body)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1 (image should be extracted once)", len(got))
	}
	if got[0] != "img-only" {
		t.Errorf("got[0] = %q, want %q", got[0], "img-only")
	}
}

func TestExtractAttachmentIDs_AllFourPatterns(t *testing.T) {
	t.Parallel()

	body := `<p><img src="/attachments/html-img-1" alt="画像1"></p>` +
		`<p><a href="/attachments/html-link-1">リンク1</a></p>` +
		`![Markdown画像](/attachments/md-img-1)` +
		`[Markdownリンク](/attachments/md-link-1)`

	got := ExtractAttachmentIDs(body)

	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want 4, got: %v", len(got), got)
	}

	want := map[string]bool{
		"html-img-1":  true,
		"html-link-1": true,
		"md-img-1":    true,
		"md-link-1":   true,
	}
	for _, id := range got {
		if !want[id] {
			t.Errorf("unexpected ID: %q", id)
		}
	}
}

func TestExtractAttachmentIDs_Deduplication(t *testing.T) {
	t.Parallel()

	body := `<p><img src="/attachments/dup-id-1" alt="1"></p>` +
		`<p><img src="/attachments/dup-id-1" alt="2"></p>` +
		`![画像](/attachments/dup-id-1)`

	got := ExtractAttachmentIDs(body)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1 (duplicates should be removed), got: %v", len(got), got)
	}
	if got[0] != "dup-id-1" {
		t.Errorf("got[0] = %q, want %q", got[0], "dup-id-1")
	}
}

func TestExtractAttachmentIDs_EmptyInput(t *testing.T) {
	t.Parallel()

	got := ExtractAttachmentIDs("")

	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}

func TestExtractAttachmentIDs_NoAttachments(t *testing.T) {
	t.Parallel()

	body := `<p>テキストのみ</p><p><img src="https://example.com/image.png"></p>` +
		`<p><a href="https://example.com">外部リンク</a></p>` +
		`![外部画像](https://example.com/img.png)`

	got := ExtractAttachmentIDs(body)

	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0, got: %v", len(got), got)
	}
}

func TestExtractAttachmentIDs_SingleQuoteAttributes(t *testing.T) {
	t.Parallel()

	body := `<img src='/attachments/single-quote-id' alt='test'>`
	got := ExtractAttachmentIDs(body)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0] != "single-quote-id" {
		t.Errorf("got[0] = %q, want %q", got[0], "single-quote-id")
	}
}

func TestExtractAttachmentIDs_SubPathNotMatched(t *testing.T) {
	t.Parallel()

	body := `<img src="/attachments/abc/extra" alt="test">` +
		`![画像](/attachments/def/extra)`

	got := ExtractAttachmentIDs(body)

	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0 (sub-paths should not match), got: %v", len(got), got)
	}
}

func TestExtractAttachmentIDs_MarkdownImageAndLinkMixed(t *testing.T) {
	t.Parallel()

	body := `![画像A](/attachments/img-a)
[リンクB](/attachments/link-b)
![画像C](/attachments/img-c)`

	got := ExtractAttachmentIDs(body)

	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3, got: %v", len(got), got)
	}

	want := map[string]bool{
		"img-a":  true,
		"link-b": true,
		"img-c":  true,
	}
	for _, id := range got {
		if !want[id] {
			t.Errorf("unexpected ID: %q", id)
		}
	}
}

func TestExtractFeaturedImageID_MarkdownImage(t *testing.T) {
	t.Parallel()

	body := "![サムネイル画像](/attachments/abc-123-def)\nテキスト"
	got := ExtractFeaturedImageID(body)

	if got == nil {
		t.Fatal("got nil, want non-nil")
	}
	if *got != "abc-123-def" {
		t.Errorf("got %q, want %q", *got, "abc-123-def")
	}
}

func TestExtractFeaturedImageID_HTMLImg(t *testing.T) {
	t.Parallel()

	body := `<img src="/attachments/xyz-456-uvw" alt="画像">` + "\nテキスト"
	got := ExtractFeaturedImageID(body)

	if got == nil {
		t.Fatal("got nil, want non-nil")
	}
	if *got != "xyz-456-uvw" {
		t.Errorf("got %q, want %q", *got, "xyz-456-uvw")
	}
}

func TestExtractFeaturedImageID_MarkdownPriorityOverHTML(t *testing.T) {
	t.Parallel()

	body := `![md画像](/attachments/md-id) <img src="/attachments/html-id">`
	got := ExtractFeaturedImageID(body)

	if got == nil {
		t.Fatal("got nil, want non-nil")
	}
	if *got != "md-id" {
		t.Errorf("got %q, want %q (Markdown should take priority)", *got, "md-id")
	}
}

func TestExtractFeaturedImageID_NoImageOnFirstLine(t *testing.T) {
	t.Parallel()

	body := "テキストのみ\n![画像](/attachments/second-line)"
	got := ExtractFeaturedImageID(body)

	if got != nil {
		t.Errorf("got %q, want nil (image is on second line)", *got)
	}
}

func TestExtractFeaturedImageID_EmptyBody(t *testing.T) {
	t.Parallel()

	got := ExtractFeaturedImageID("")

	if got != nil {
		t.Errorf("got %q, want nil", *got)
	}
}

func TestExtractFeaturedImageID_WhitespaceFirstLine(t *testing.T) {
	t.Parallel()

	body := "  ![画像](/attachments/ws-id)  \nテキスト"
	got := ExtractFeaturedImageID(body)

	if got == nil {
		t.Fatal("got nil, want non-nil")
	}
	if *got != "ws-id" {
		t.Errorf("got %q, want %q", *got, "ws-id")
	}
}

func TestExtractFeaturedImageID_BlankFirstLine(t *testing.T) {
	t.Parallel()

	body := "  \n![画像](/attachments/second-line)"
	got := ExtractFeaturedImageID(body)

	if got != nil {
		t.Errorf("got %q, want nil (first line is blank)", *got)
	}
}

func TestExtractFeaturedImageID_LinkNotImage(t *testing.T) {
	t.Parallel()

	body := "[リンク](/attachments/link-id)\nテキスト"
	got := ExtractFeaturedImageID(body)

	if got != nil {
		t.Errorf("got %q, want nil (link is not an image)", *got)
	}
}

func TestExtractFeaturedImageID_HTMLImgCaseInsensitive(t *testing.T) {
	t.Parallel()

	body := `<IMG SRC="/attachments/upper-case-id" ALT="test">`
	got := ExtractFeaturedImageID(body)

	if got == nil {
		t.Fatal("got nil, want non-nil")
	}
	if *got != "upper-case-id" {
		t.Errorf("got %q, want %q", *got, "upper-case-id")
	}
}

func TestExtractFeaturedImageID_EmptyAlt(t *testing.T) {
	t.Parallel()

	body := "![](/attachments/empty-alt-id)"
	got := ExtractFeaturedImageID(body)

	if got == nil {
		t.Fatal("got nil, want non-nil")
	}
	if *got != "empty-alt-id" {
		t.Errorf("got %q, want %q", *got, "empty-alt-id")
	}
}
