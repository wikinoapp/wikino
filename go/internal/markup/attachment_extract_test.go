package markup

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestScanAttachmentRefMatches_ParsedLabelSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		label string
	}{
		{name: "HTML 属性の閉じ角括弧", label: `<span title="]">label</span>`},
		{name: "HTML 属性の開き角括弧", label: `<span title="[">label</span>`},
		{name: "HTML コメントの閉じ角括弧", label: `<!-- ] -->label`},
		{name: "HTML コメントの開き角括弧", label: `<!-- [ -->label`},
		{name: "子画像のリンク先の閉じ角括弧", label: `![image](https://example.com/image]x.png)`},
		{name: "子画像のリンク先の開き角括弧", label: `![image](https://example.com/image[x.png)`},
		{name: "子画像のタイトルの閉じ角括弧", label: `![image](https://example.com/image.png "]")`},
		{name: "子画像のタイトルの開き角括弧", label: `![image](https://example.com/image.png "[")`},
		{name: "子画像の中の HTML", label: `![<span title="]">image</span>](https://example.com/image.png)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := "[" + tt.label + "](/attachments/01ABC)"
			matches := ScanAttachmentRefMatches(body)
			if len(matches) != 1 {
				t.Fatalf("matches = %+v, want one attachment", matches)
			}
			match := matches[0]
			if match.AttachmentID != "01ABC" || match.InHTMLAttribute || body[match.Start:match.Stop] != "/attachments/01ABC" {
				t.Errorf("match = %+v, want Markdown destination /attachments/01ABC", match)
			}
		})
	}
}

func TestScanAttachmentRefMatches_DestinationSyntax(t *testing.T) {
	t.Parallel()

	type want struct {
		destination     string
		attachmentID    model.AttachmentID
		inHTMLAttribute bool
	}
	tests := []struct {
		name string
		body string
		want []want
	}{
		{
			name: "引用内で改行したリンク先",
			body: "> [file](\n>   /attachments/01ABC)",
			want: []want{{destination: "/attachments/01ABC", attachmentID: "01ABC"}},
		},
		{
			name: "入れ子の引用とリスト内で改行した山括弧付き画像",
			body: "> > - ![image](\n> >   </attachments/01ABC>\n> >   \"title\")",
			want: []want{{destination: "/attachments/01ABC", attachmentID: "01ABC"}},
		},
		{
			name: "引用内で改行した参照定義",
			body: "> [file][ref]\n>\n> [ref]:\n>   /attachments/01ABC",
			want: []want{{destination: "/attachments/01ABC", attachmentID: "01ABC"}},
		},
		{
			name: "リスト内で改行した画像の参照定義",
			body: "- ![image][ref]\n\n  [ref]:\n    </attachments/01ABC> \"title\"",
			want: []want{{destination: "/attachments/01ABC", attachmentID: "01ABC"}},
		},
		{
			name: "子画像のタイトルを飛ばして外側も返す",
			body: `[![image](/attachments/01ABC "]")](/attachments/01DEF)`,
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
				{destination: "/attachments/01DEF", attachmentID: "01DEF"},
			},
		},
		{
			name: "引用符なしの img と a 属性",
			body: `<img alt=image src=/attachments/01ABC width=10> <a href=/attachments/01DEF>file</a>`,
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC", inHTMLAttribute: true},
				{destination: "/attachments/01DEF", attachmentID: "01DEF", inHTMLAttribute: true},
			},
		},
		{
			name: "引用符なしでも data 属性とコメントと Markdown のコードを除外",
			body: "<img data-src=/attachments/01ABC> <a data-href=/attachments/01ABC>file</a>\n\n" +
				"<!-- <img src=/attachments/01ABC> -->\n\n`<img src=/attachments/01ABC>`\n\n" +
				"```html\n<a href=/attachments/01ABC>file</a>\n```",
		},
		{
			name: "raw HTML の code 要素の中の参照は返す",
			body: "<code><img src=/attachments/01ABC></code>",
			want: []want{{destination: "/attachments/01ABC", attachmentID: "01ABC", inHTMLAttribute: true}},
		},
		{
			name: "別属性の値にある src を除外",
			body: `<img title=" src=/attachments/01ABC" src=/attachments/01DEF>`,
			want: []want{{destination: "/attachments/01DEF", attachmentID: "01DEF", inHTMLAttribute: true}},
		},
		{
			name: "後続属性にあるタグの例を除外",
			body: `<img src=/attachments/01ABC title='<img src=/attachments/01DEF>'>`,
			want: []want{{destination: "/attachments/01ABC", attachmentID: "01ABC", inHTMLAttribute: true}},
		},
		{
			name: "Markdown のパーセントエンコード",
			body: `[file](/attachments/%30%31ABC)`,
			want: []want{{destination: "/attachments/%30%31ABC", attachmentID: "01ABC"}},
		},
		{
			name: "Markdown の文字参照とエスケープ",
			body: `![image](/attachments/&#48;\%31ABC)`,
			want: []want{{destination: `/attachments/&#48;\%31ABC`, attachmentID: "01ABC"}},
		},
		{
			name: "改行した参照定義の文字参照とパーセントエンコード",
			body: "> [file][ref]\n>\n> [ref]:\n>   </attachments/&#x30;%31ABC>",
			want: []want{{destination: "/attachments/&#x30;%31ABC", attachmentID: "01ABC"}},
		},
		{
			name: "HTML 属性の文字参照",
			body: `<img src="/attachments/&#48;1ABC"> <a href='/attachments/&#x30;%31ABC'>file</a>`,
			want: []want{
				{destination: "/attachments/&#48;1ABC", attachmentID: "01ABC", inHTMLAttribute: true},
				{destination: "/attachments/&#x30;%31ABC", attachmentID: "01ABC", inHTMLAttribute: true},
			},
		},
		{
			name: "引用符なしの HTML 属性の名前付き文字参照",
			body: `<img src=/attachments/&percnt;30%31ABC>`,
			want: []want{{destination: "/attachments/&percnt;30%31ABC", attachmentID: "01ABC", inHTMLAttribute: true}},
		},
		{
			name: "パーセントエンコードを二重に復号しない",
			body: `[file](/attachments/%2530%2531ABC) <img src="/attachments/%2530%2531ABC">`,
			want: []want{
				{destination: "/attachments/%2530%2531ABC", attachmentID: "%30%31ABC"},
				{destination: "/attachments/%2530%2531ABC", attachmentID: "%30%31ABC", inHTMLAttribute: true},
			},
		},
		{
			name: "HTML の文字参照を二重に復号しない",
			body: `<img src="/attachments/&amp;#48;1ABC">`,
			want: []want{{destination: "/attachments/&amp;#48;1ABC", attachmentID: "&#48;1ABC", inHTMLAttribute: true}},
		},
		{
			name: "HTML 属性のセミコロンなしの曖昧な文字参照はそのまま",
			body: `<img src="/attachments/A&timesx">`,
			want: []want{{destination: "/attachments/A&timesx", attachmentID: "A&timesx", inHTMLAttribute: true}},
		},
		{
			name: "不正なパーセントエンコードを拒否",
			body: `[file](/attachments/%3ZABC) <img src=/attachments/01ABC%>`,
		},
		{
			name: "復号後のスラッシュとバックスラッシュを拒否",
			body: `[file](/attachments/01%2FABC) ![image](/attachments/01%5cABC) <img src="/attachments/01&#47;ABC"> <a href=/attachments/01%5CABC>file</a>`,
		},
		{
			name: "HTML 属性で Markdown のエスケープを解決しない",
			body: `<img src="/attachments/\%30%31ABC">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matches := ScanAttachmentRefMatches(tt.body)
			if len(matches) != len(tt.want) {
				t.Fatalf("matches = %+v, want %d references", matches, len(tt.want))
			}
			previousStop := 0
			for i, match := range matches {
				want := tt.want[i]
				if match.Start < previousStop || match.Stop <= match.Start || match.Stop > len(tt.body) {
					t.Fatalf("invalid source range: %+v", match)
				}
				if tt.body[match.Start:match.Stop] != want.destination || match.AttachmentID != want.attachmentID || match.InHTMLAttribute != want.inHTMLAttribute {
					t.Errorf("match[%d] = %+v (%q), want %+v", i, match, tt.body[match.Start:match.Stop], want)
				}
				previousStop = match.Stop
			}
		})
	}
}

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

	// 各記法は空行で区切る。区切らないと HTML ブロックが行末まで続き、その中の Markdown 記法は
	// 画面上でも文字のまま出るため参照にならない。
	body := "<p><img src=\"/attachments/html-img-1\" alt=\"画像1\"></p>\n\n" +
		"<p><a href=\"/attachments/html-link-1\">リンク1</a></p>\n\n" +
		"![Markdown画像](/attachments/md-img-1)\n\n" +
		"[Markdownリンク](/attachments/md-link-1)"

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

func TestExtractAttachmentIDs_MarkdownImageWithTitle(t *testing.T) {
	t.Parallel()

	body := `![桜開花のお知らせ](/attachments/01990988-2b4a-8777-57f0-8cd72decd1fd "桜開花のお知らせ")`
	got := ExtractAttachmentIDs(body)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0] != "01990988-2b4a-8777-57f0-8cd72decd1fd" {
		t.Errorf("got[0] = %q, want %q", got[0], "01990988-2b4a-8777-57f0-8cd72decd1fd")
	}
}

func TestExtractAttachmentIDs_MarkdownLinkWithTitle(t *testing.T) {
	t.Parallel()

	body := `[ダウンロード](/attachments/abc-123-def "ファイル名")`
	got := ExtractAttachmentIDs(body)

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0] != "abc-123-def" {
		t.Errorf("got[0] = %q, want %q", got[0], "abc-123-def")
	}
}

func TestExtractAttachmentIDs_PercentEncodedBackslash(t *testing.T) {
	t.Parallel()

	// bluemondayがバックスラッシュを%5Cにエンコードしたケース
	body := `<img src="/attachments/att-1%5C">`
	got := ExtractAttachmentIDs(body)

	if len(got) != 0 {
		t.Errorf("percent-encoded backslash ID should be excluded, got: %v", got)
	}
}

func TestExtractFeaturedImageID_MarkdownImageWithTitle(t *testing.T) {
	t.Parallel()

	body := `![桜開花のお知らせ](/attachments/feat-title-id "タイトル")` + "\nテキスト"
	got := ExtractFeaturedImageID(body)

	if got == nil {
		t.Fatal("got nil, want non-nil")
	}
	if *got != "feat-title-id" {
		t.Errorf("got %q, want %q", *got, "feat-title-id")
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

func TestScanAttachmentRefMatches(t *testing.T) {
	t.Parallel()

	type want struct {
		destination     string
		attachmentID    model.AttachmentID
		inHTMLAttribute bool
	}

	tests := []struct {
		name string
		body string
		want []want
	}{
		{
			name: "Markdown の画像のリンク先を位置とともに返す",
			body: "![図](/attachments/01ABC)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "Markdown のリンクのリンク先を位置とともに返す",
			body: "[資料](/attachments/01ABC)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "img 要素の src は HTML の属性値として返す",
			body: `<img alt="図" src="/attachments/01ABC">`,
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC", inHTMLAttribute: true},
			},
		},
		{
			name: "a 要素の href は HTML の属性値として返す",
			body: `<a href='/attachments/01ABC'>資料</a>`,
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC", inHTMLAttribute: true},
			},
		},
		{
			name: "タイトル付きのリンクではリンク先だけを返す",
			body: `![図](/attachments/01ABC "図の説明")`,
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "複数の参照を現れる順に返す",
			body: "![図](/attachments/01ABC)\n\n[資料](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			// エディタは画像以外のファイルのアップロードでファイル名の角括弧をエスケープする
			name: "ラベルにエスケープした角括弧を含むリンクのリンク先を返す",
			body: `[報告書\]2026.pdf](/attachments/01ABC)`,
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "ラベルのコードスパンに角括弧を含むリンクのリンク先を返す",
			body: "[コード `]` の図](/attachments/01ABC)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "ラベルのコードスパンに角括弧を含む画像のリンク先を返す",
			body: "![`]`図](/attachments/01ABC)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "入れ子の画像リンクの内側と外側のリンク先を現れる順に返す",
			body: "[![図](/attachments/01ABC)](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "山括弧で囲んだリンク先を返す",
			body: "[資料](</attachments/01ABC>)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "参照リンクのリンク先を定義側の位置とともに返す",
			body: "[資料][ref]\n\n[ref]: /attachments/01ABC",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "参照画像のリンク先を定義側の位置とともに返す",
			body: "![図][ref]\n\n[ref]: /attachments/01ABC",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "同じ定義を複数のリンクが指していても定義側の 1 件だけを返す",
			body: "[資料][ref] と [別の資料][ref]\n\n[ref]: /attachments/01ABC",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "タイトル付きのリンク参照定義ではリンク先だけを返す",
			body: `[資料][ref]` + "\n\n" + `[ref]: /attachments/01ABC "説明"`,
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "山括弧で囲んだリンク参照定義のリンク先を返す",
			body: "[資料][ref]\n\n[ref]: </attachments/01ABC>",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "フェンス付きコードブロックの中の参照は返さない",
			body: "```\n![図](/attachments/01ABC)\n```\n\n![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "フェンス付きコードブロックの情報文字列にある参照は返さない",
			body: "```![図](/attachments/01ABC)\n本文\n```\n\n![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "インラインコードの中の参照は返さない",
			body: "`![図](/attachments/01ABC)` と ![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "字下げされたコードブロックの中の参照は返さない",
			body: "段落。\n\n    ![図](/attachments/01ABC)\n\n![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "行頭の pre 要素が開く HTML ブロックの中の参照は返さない",
			body: "<pre>\n![図](/attachments/01ABC)\n</pre>\n\n![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "自己終了記法の code 要素の中と後ろにある参照を返す",
			body: "<code/>![図](/attachments/01ABC)</code> と ![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "段落中の code 要素の中にある参照を返す",
			body: "本文 <code>[file](/attachments/01ABC)</code> 本文",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "自己終了記法の pre 要素が開く HTML ブロックの中の参照は返さない",
			body: "<pre/>![図](/attachments/01ABC)</pre>\n\n![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "段落中の pre 要素の中にある参照を返す",
			body: "本文 <pre/>![図](/attachments/01ABC)</pre> 本文",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "HTML として解釈されない img 風の文字列は返さない",
			body: `\<img src="/attachments/01ABC">` + "\n\n" +
				`<!-- <img src="/attachments/01DEF"> -->` + "\n\n" +
				`<script>const image = '<img src="/attachments/01GHI">'</script>` + "\n\n" +
				`<style>.x { content: '<img src="/attachments/01JKL">' }</style>` + "\n\n" +
				`<img src="/attachments/01XYZ">`,
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ", inHTMLAttribute: true},
			},
		},
		{
			name: "閉じ角括弧の無いタグは後続のタグと 1 つに読まれても返さない",
			body: `<img src="/attachments/01ABC" alt="図"` + "\nキャプション <br>\n\n" +
				`<a href="/attachments/01DEF" ラベル</a>` + "\n\n" +
				`<img src="/attachments/01XYZ"> <br>`,
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ", inHTMLAttribute: true},
			},
		},
		{
			name: "ラベルの中で始まる破棄要素の後ろにあるリンク先を返す",
			body: "[<title>x](/attachments/01ABC)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "ラベルの中で始まる raw text 要素の後ろにあるリンク先を返す",
			body: "![<textarea>x](/attachments/01ABC)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC"},
			},
		},
		{
			name: "破棄要素の中で始まるリンクのリンク先は返さない",
			body: "<title>[x](/attachments/01ABC)</title>",
			want: nil,
		},
		{
			name: "段落中の raw text 要素の中の参照は返さない",
			body: "本文 <textarea>![図](/attachments/01ABC)</textarea> 本文\n\n" +
				"本文 <xmp>![図](/attachments/01DEF)</xmp> 本文\n\n" +
				"![図](/attachments/01XYZ)\n\n" +
				"本文 <plaintext>![図](/attachments/01GHI)</plaintext> 本文",
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "通常の文章にある src と Markdown の断片は返さない",
			body: `example src="/attachments/01ABC" and ](/attachments/01DEF) then ![図](/attachments/01XYZ)`,
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "data-src 属性は返さない",
			body: `<div data-src="/attachments/01ABC"></div>` + "\n\n![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "対象外要素の src と href 属性は返さない",
			body: `<video src="/attachments/01ABC"></video><link href="/attachments/01DEF">` + "\n\n![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			// サニタイズが中身ごと落とす要素の参照は表示側の ExtractAttachmentIDs も拾わず、
			// アーカイブに複製が入らないため、書き換えると存在しないファイルを指すことになる
			name: "サニタイズが中身ごと落とす要素の中の参照は返さない",
			body: `<object><img src="/attachments/01ABC"></object>` + "\n\n" +
				`<frameset><a href="/attachments/01DEF">資料</a></frameset>` + "\n\n" +
				`![図](/attachments/01XYZ)`,
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "破棄要素を異なる破棄要素の終了タグで閉じた後の参照は返す",
			body: `<object><img src="/attachments/01ABC"></iframe>` + "\n\n" +
				`![図](/attachments/01XYZ)`,
			want: []want{
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "破棄要素の中の script 要素の終了タグは破棄を終わらせない",
			body: `<object><img src="/attachments/01ABC"></script>` + "\n\n" +
				`![図](/attachments/01XYZ)`,
			want: nil,
		},
		{
			name: "入れ子の a 要素の中の参照も現れる順に返す",
			body: `<a href="/attachments/01ABC"><a href="/attachments/01XYZ"></a>` + "\n\n" +
				`![図](/attachments/01DEF)`,
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC", inHTMLAttribute: true},
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ", inHTMLAttribute: true},
				{destination: "/attachments/01DEF", attachmentID: "01DEF"},
			},
		},
		{
			// img タグだけの行は HTML ブロックを開かないため、続く行も画面と同じく Markdown として
			// 読まれる
			name: "img タグの行に続く Markdown の参照も返す",
			body: `<img src="/attachments/01ABC">` + "\n*キャプション*\n![図](/attachments/01XYZ)",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC", inHTMLAttribute: true},
				{destination: "/attachments/01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "img タグの行に続くコードの中の参照は返さない",
			body: `<img src="/attachments/01ABC">` + "\n*キャプション*\n`![図](/attachments/01XYZ)`",
			want: []want{
				{destination: "/attachments/01ABC", attachmentID: "01ABC", inHTMLAttribute: true},
			},
		},
		{
			name: "パスの区切りを文字参照で書いた参照も返す",
			body: "[a](&#47;attachments/01ABC) と [b](/attachments&#x2F;01XYZ)",
			want: []want{
				{destination: "&#47;attachments/01ABC", attachmentID: "01ABC"},
				{destination: "/attachments&#x2F;01XYZ", attachmentID: "01XYZ"},
			},
		},
		{
			name: "パスの区切りをエスケープで書いた参照も返す",
			body: `[a](\/attachments/01ABC)`,
			want: []want{
				{destination: `\/attachments/01ABC`, attachmentID: "01ABC"},
			},
		},
		{
			name: "HTML 属性のパスの区切りを文字参照で書いた参照も返す",
			body: `<img src="&#47;attachments/01ABC">`,
			want: []want{
				{destination: "&#47;attachments/01ABC", attachmentID: "01ABC", inHTMLAttribute: true},
			},
		},
		{
			name: "添付ファイルを参照しない本文では何も返さない",
			body: "[外部リンク](https://example.com/) と本文。",
			want: nil,
		},
		{
			name: "エスケープ記号があっても添付ファイルを参照しなければ何も返さない",
			body: `[外部リンク](https://example.com/?a=1&b=2) と \[本文\]。`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matches := ScanAttachmentRefMatches(tt.body)

			if len(matches) != len(tt.want) {
				t.Fatalf("len(matches) = %d, want %d", len(matches), len(tt.want))
			}

			for i, w := range tt.want {
				got := matches[i]

				if destination := tt.body[got.Start:got.Stop]; destination != w.destination {
					t.Errorf("matches[%d] の範囲の文字列 = %q, want %q", i, destination, w.destination)
				}
				if got.AttachmentID != w.attachmentID {
					t.Errorf("matches[%d].AttachmentID = %q, want %q", i, got.AttachmentID, w.attachmentID)
				}
				if got.InHTMLAttribute != w.inHTMLAttribute {
					t.Errorf("matches[%d].InHTMLAttribute = %t, want %t", i, got.InHTMLAttribute, w.inHTMLAttribute)
				}
			}
		})
	}
}
