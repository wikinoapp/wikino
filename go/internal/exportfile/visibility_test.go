package exportfile

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestRewriteAttachmentLinks_ReferenceVisibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		live bool
	}{
		{name: "definition inside object", body: "[visible][r]\n\nx <object>\n\n[r]: /attachments/01A\n\n</object>", live: true},
		{name: "definition inside textarea", body: "[visible][r]\n\nx <textarea>\n\n[r]: /attachments/01A\n\n</textarea>", live: true},
		{name: "only use inside object", body: "x <object>[hidden][r]</object>\n\n[r]: /attachments/01A"},
		{name: "only use inside textarea", body: "x <textarea>[hidden][r]</textarea>\n\n[r]: /attachments/01A"},
		{name: "hidden use before visible use", body: "x <object>[hidden][r]</object> [visible][r]\n\n[r]: /attachments/01A", live: true},
		{name: "visible use before hidden use", body: "[visible][r] <textarea>[hidden][r]</textarea>\n\n[r]: /attachments/01A", live: true},
		{name: "reference images share one definition", body: "![r] ![r][] ![image][r]\n\n[r]: /attachments/01A", live: true},
		{name: "reference links share one definition", body: "[r] [r][] [file][r]\n\n[r]: /attachments/01A", live: true},
		{name: "normalized label", body: "[file][ STRASSE  label ]\n\n[Straße label]: /attachments/01A", live: true},
		{name: "first definition wins", body: "[file][r]\n\n[r]: /attachments/01A\n[r]: /attachments/01B", live: true},
		{name: "shadowed attachment definition", body: "[file][r]\n\n[r]: /elsewhere\n[r]: /attachments/01A"},
		{name: "unused definition", body: "[r]: /attachments/01A"},
		{name: "code is not a use", body: "`[file][r]`\n\n[r]: /attachments/01A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rendered := markup.RenderMarkdown(tt.body)
			live := strings.Contains(rendered, `href="/attachments/01A"`) || strings.Contains(rendered, `src="/attachments/01A"`)
			if live != tt.live {
				t.Fatalf("rendered reference visibility = %v, want %v: %s", live, tt.live, rendered)
			}
			var wantIDs []string
			want := tt.body
			if tt.live {
				wantIDs = []string{"01A"}
				want = strings.Replace(tt.body, "/attachments/01A", "attachments/a.pdf", 1)
			}
			if ids := markup.ExtractAttachmentIDs(tt.body); !slices.Equal(ids, wantIDs) {
				t.Errorf("ExtractAttachmentIDs() = %v, want %v", ids, wantIDs)
			}
			matches := markup.ScanAttachmentRefMatches(tt.body)
			if len(matches) != len(wantIDs) {
				t.Fatalf("matches = %+v, want %d", matches, len(wantIDs))
			}
			if tt.live {
				match := matches[0]
				if match.Start != strings.Index(tt.body, "/attachments/01A") ||
					match.Stop != match.Start+len("/attachments/01A") || match.InHTMLAttribute {
					t.Errorf("match = %+v, want definition destination", match)
				}
			}
			names := map[model.AttachmentID]string{"01A": "a.pdf", "01B": "b.pdf"}
			if got := RewriteAttachmentLinks(tt.body, names); got != want {
				t.Errorf("RewriteAttachmentLinks() = %q, want %q", got, want)
			}
		})
	}
}

func TestRewriteLinks_AfterRejectedRawTextEndTags(t *testing.T) {
	t.Parallel()

	for _, element := range []string{"textarea", "xmp", "script", "style", "title", "iframe", "noembed", "noscript"} {
		for _, fakeElement := range []string{"object", "code"} {
			t.Run(element+"/"+fakeElement, func(t *testing.T) {
				t.Parallel()

				body := fmt.Sprintf("x <%s>`</%s>` <%s> `</%s>` </%s> <img src=\"/attachments/01B\"> [[A]]",
					element, element, fakeElement, element, element)
				rendered := markup.RenderMarkdown(body)
				if !strings.Contains(rendered, `<img src="/attachments/01B">`) {
					t.Fatalf("rendered image is missing: %s", rendered)
				}
				converted := markup.ReplaceWikilinks(rendered, "T", "space", []markup.PageLocation{
					{Key: markup.WikilinkKey{Raw: "A"}, TopicName: "T", PageNumber: 1, PageTitle: "A"},
				})
				if !strings.Contains(converted, `href="/s/space/pages/1"`) {
					t.Fatalf("rendered wiki link is missing: %s", converted)
				}
				wikiMatches := markup.ScanWikilinkMatches(body, "T")
				if len(wikiMatches) != 1 || wikiMatches[0].Key.PageTitle != "A" {
					t.Errorf("wiki matches = %+v, want A", wikiMatches)
				}
				attachmentMatches := markup.ScanAttachmentRefMatches(body)
				if len(attachmentMatches) != 1 || attachmentMatches[0].AttachmentID != "01B" {
					t.Errorf("attachment matches = %+v, want 01B", attachmentMatches)
				}
				if got := RewriteWikilinks(body, "T", map[PageRef]string{{TopicName: "T", PageTitle: "A"}: "T/A.md"}); got != strings.Replace(body, "[[A]]", "[[T/A.md]]", 1) {
					t.Errorf("unexpected wiki-link rewrite: %q", got)
				}
				if got := RewriteAttachmentLinks(body, map[model.AttachmentID]string{"01B": "b.png"}); got != strings.Replace(body, "/attachments/01B", "attachments/b.png", 1) {
					t.Errorf("unexpected attachment rewrite: %q", got)
				}
			})
		}
	}
}

func TestRewriteWikilinks_TableAndMarkdownContainers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "table closes implicit paragraph", body: "<code><table><tr><td>[[A]]</td></tr></table></code>", want: []string{"A"}},
		{name: "table restores code after closing", body: "<code><table><tr><td>[[A]]</td></tr></table>[[B]]</code>", want: []string{"A"}},
		{name: "raw block keeps code around table", body: "<div><code><table><tr><td>[[A]]</td></tr></table>[[B]]</code></div>"},
		{name: "pre keeps table protected", body: "<pre><table><tr><td>[[A]]</td></tr></table>[[B]]</pre>"},
		{name: "foster-parented code leaves cell visible", body: "<table><code><tr><td>[[A]]</td></tr></table>[[B]]", want: []string{"A"}},
		{name: "Markdown table inside reconstructed code", body: "x <code>\n\n| [[A]] |\n| --- |\n\n[[B]]"},
		{name: "raw end tag closes Markdown quote", body: "> <pre>\n> </blockquote>[[A]]", want: []string{"A"}},
		{name: "raw end tag closes Markdown list item", body: "- <pre>\n  </li>[[A]]", want: []string{"A"}},
		{name: "quote marker inside pre is text", body: "> <pre>\n> > </blockquote>[[A]]", want: []string{"A"}},
		{name: "whitespace between table and row", body: "<code><table>\n  <tr><td>[[A]]</td></tr></table></code>", want: []string{"A"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			paths := map[PageRef]string{}
			var locations []markup.PageLocation
			for i, title := range []string{"A", "B"} {
				paths[PageRef{TopicName: "T", PageTitle: title}] = "T/" + title + ".md"
				locations = append(locations, markup.PageLocation{
					Key: markup.WikilinkKey{Raw: title}, TopicName: "T", PageNumber: i + 1, PageTitle: title,
				})
			}
			rendered := markup.ReplaceWikilinks(markup.RenderMarkdown(tt.body), "T", "space", locations)
			for i, title := range []string{"A", "B"} {
				if got := strings.Contains(rendered, fmt.Sprintf(`href="/s/space/pages/%d"`, i+1)); got != slices.Contains(tt.want, title) {
					t.Fatalf("unexpected rendered visibility for %s: %s", title, rendered)
				}
			}
			var titles []string
			for _, match := range markup.ScanWikilinkMatches(tt.body, "T") {
				titles = append(titles, match.Key.PageTitle)
				if tt.body[match.Start:match.Stop] != "[["+match.Key.PageTitle+"]]" {
					t.Errorf("incorrect source range: %+v", match)
				}
			}
			if !slices.Equal(titles, tt.want) {
				t.Errorf("scanned titles = %v, want %v", titles, tt.want)
			}
			want := tt.body
			for _, title := range tt.want {
				want = strings.ReplaceAll(want, "[["+title+"]]", "[[T/"+title+".md]]")
			}
			if got := RewriteWikilinks(tt.body, "T", paths); got != want {
				t.Errorf("RewriteWikilinks() = %q, want %q", got, want)
			}
		})
	}
}

func TestRewriteWikilinks_RendererDepthLimit(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("<code>", 600) + strings.Repeat("</code>", 600) + "[[A]]"
	rendered := markup.RenderMarkdown(body)
	converted := markup.ReplaceWikilinks(rendered, "T", "space", []markup.PageLocation{
		{Key: markup.WikilinkKey{Raw: "A"}, TopicName: "T", PageNumber: 1, PageTitle: "A"},
	})
	if converted != rendered {
		t.Fatalf("display converted a body beyond its parser's depth limit")
	}
	if matches := markup.ScanWikilinkMatches(body, "T"); len(matches) != 0 {
		t.Errorf("matches = %+v, want no conversion beyond the display depth limit", matches)
	}
	if got := RewriteWikilinks(body, "T", map[PageRef]string{{TopicName: "T", PageTitle: "A"}: "T/A.md"}); got != body {
		t.Errorf("export changed a body display cannot convert: %q", got)
	}
}

func TestRewriteWikilinks_CodeAfterRawTextElement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "code span", body: "`[[A]]`"},
		{name: "fenced block", body: "```\n[[A]]\n```"},
		{name: "indented block", body: "    [[A]]\n"},
		{name: "fence language", body: "```[[A]]\nx\n```"},
		{name: "fence after language", body: "```x [[A]]\ny\n```"},
		{name: "xmp opens code span", body: "<xmp>`[[A]]`", want: []string{"A"}},
		{name: "xmp opens fenced block", body: "<xmp>\n\n```\n[[A]]\n```", want: []string{"A"}},
		{name: "xmp opens indented block", body: "<xmp>\n\n    [[A]]\n", want: []string{"A"}},
		{name: "xmp opens fence language", body: "<xmp>\n\n```[[A]]\nx\n```", want: []string{"A"}},
		{name: "xmp drops text after fence language", body: "<xmp>\n\n```x [[A]]\ny\n```"},
		{name: "plaintext opens code span", body: "<plaintext>`[[A]]`", want: []string{"A"}},
		{name: "textarea opens code span", body: "<textarea>x\n\n`[[A]]`", want: []string{"A"}},
		{name: "closed xmp restores code", body: "<xmp>y</xmp>\n\n`[[A]]`"},
		{name: "code span still splits text", body: "[[ と書く `x` [[A]]", want: []string{"A"}},
		{name: "code span splits a pair apart", body: "[[A`x`]]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertWikilinkParity(t, tt.body, tt.want)
		})
	}
}

// assertWikilinkParity checks that display, the scan and the rewrite agree on which wiki links of
// body reach the reader. want holds the page titles the screen turns into links.
//
// [Ja] assertWikilinkParity は、body のどの Wiki リンクが読み手に届くかについて、表示・走査・
// 書き換えが一致することを確かめる。want は画面がリンクに変える (べき) ページタイトル。
func assertWikilinkParity(t *testing.T, body string, want []string) {
	t.Helper()

	titles := []string{"A", "B"}
	paths := map[PageRef]string{}
	var locations []markup.PageLocation
	for i, title := range titles {
		paths[PageRef{TopicName: "T", PageTitle: title}] = "T/" + title + ".md"
		locations = append(locations, markup.PageLocation{
			Key: markup.WikilinkKey{Raw: title}, TopicName: "T", PageNumber: i + 1, PageTitle: title,
		})
	}

	rendered := markup.ReplaceWikilinks(markup.RenderMarkdown(body), "T", "space", locations)
	for i, title := range titles {
		got := strings.Contains(rendered, fmt.Sprintf(`href="/s/space/pages/%d"`, i+1))
		if got != slices.Contains(want, title) {
			t.Fatalf("rendered visibility for %s = %v, want %v: %s", title, got, !got, rendered)
		}
	}

	var scanned []string
	for _, match := range markup.ScanWikilinkMatches(body, "T") {
		scanned = append(scanned, match.Key.PageTitle)
		if body[match.Start:match.Stop] != "[["+match.Key.PageTitle+"]]" {
			t.Errorf("incorrect source range: %+v", match)
		}
	}
	if !slices.Equal(scanned, want) {
		t.Errorf("scanned titles = %v, want %v", scanned, want)
	}

	rewritten := body
	for _, title := range want {
		rewritten = strings.ReplaceAll(rewritten, "[["+title+"]]", "[[T/"+title+".md]]")
	}
	if got := RewriteWikilinks(body, "T", paths); got != rewritten {
		t.Errorf("RewriteWikilinks() = %q, want %q", got, rewritten)
	}
}

func TestRewriteAttachmentLinks_DroppedBySanitizer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		live bool
	}{
		{name: "unterminated comment in a quote", body: "><!--\n\n<img src=\"/attachments/01A\">"},
		{name: "unterminated comment in a cell", body: "<td><!--\n\n<img src=\"/attachments/01A\">"},
		{name: "unterminated comment before a Markdown image", body: "><!--\n\n![x](/attachments/01A)"},
		{name: "unterminated tag before a link", body: "<td\n\n<a href=\"/attachments/01A\">"},
		{name: "terminated comment leaves the image live", body: "<!-- y -->\n\n<img src=\"/attachments/01A\">", live: true},
		{name: "quote with a terminated comment", body: "> <!-- y -->\n\n<img src=\"/attachments/01A\">", live: true},
		{name: "image inside a raw code element", body: "<code><img src=\"/attachments/01A\"></code>", live: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rendered := markup.RenderMarkdown(tt.body)
			live := strings.Contains(rendered, `href="/attachments/01A"`) ||
				strings.Contains(rendered, `src="/attachments/01A"`)
			if live != tt.live {
				t.Fatalf("rendered reference visibility = %v, want %v: %s", live, tt.live, rendered)
			}

			var wantIDs []string
			want := tt.body
			if tt.live {
				wantIDs = []string{"01A"}
				want = strings.Replace(tt.body, "/attachments/01A", "attachments/a.png", 1)
			}
			if ids := markup.ExtractAttachmentIDs(tt.body); !slices.Equal(ids, wantIDs) {
				t.Errorf("ExtractAttachmentIDs() = %v, want %v", ids, wantIDs)
			}
			if got := RewriteAttachmentLinks(tt.body, map[model.AttachmentID]string{"01A": "a.png"}); got != want {
				t.Errorf("RewriteAttachmentLinks() = %q, want %q", got, want)
			}
		})
	}
}

func TestRewriteWikilinks_CommentAndDoctypeContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "unterminated comment", body: "<!--[[A]]"},
		{name: "terminated comment", body: "<!--[[A]]-->"},
		{name: "inline comment", body: "x <!-- [[A]] -->"},
		{name: "doctype", body: "x\n<!DOCTYPE [[A]]>"},
		{name: "attribute of a tag", body: `<div title="[[A]]">y</div>`},
		{name: "comment inside an unterminated pre", body: "b<pre><textarea>\n<!--[[A]]"},
		{name: "xmp turns a comment into text", body: "<xmp>\n\n<!--[[A]]", want: []string{"A"}},
		{name: "textarea turns a comment into text", body: "<textarea>\n<!--[[A]]", want: []string{"A"}},
		{name: "xmp turns a pre into text as well", body: "<xmp>\n\n<pre>\n\n<!--[[A]]", want: []string{"A"}},
		{name: "xmp turns a doctype into text", body: "<xmp>\n\n<!DOCTYPE [[A]]>", want: []string{"A"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertWikilinkParity(t, tt.body, tt.want)
		})
	}
}

func TestRewriteAttachmentLinks_DuplicateReferenceVisibility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		kept []int
	}{
		{name: "hidden raw image", body: "![visible](/attachments/01A)\n\n> <!--\n\n<img src=\"/attachments/01A\">", kept: []int{0}},
		{name: "hidden Markdown image", body: "![visible](/attachments/01A)\n\n> <!--\n\n![hidden](/attachments/01A)", kept: []int{0}},
		{name: "hidden raw anchor", body: "[visible](/attachments/01A)\n\n> <!--\n\n<a href=\"/attachments/01A\">hidden</a>", kept: []int{0}},
		{name: "hidden Markdown anchor", body: "[visible](/attachments/01A)\n\n> <!--\n\n[hidden](/attachments/01A)", kept: []int{0}},
		{name: "hidden reference definition", body: "[visible](/attachments/01A)\n\n> <!--\n\n[hidden][r]\n\n[r]: /attachments/01A", kept: []int{0}},
		{name: "shared definition with a visible use", body: "[visible][r]\n\n> <!--\n\n[hidden][r]\n\n[r]: /attachments/01A", kept: []int{0}},
		{name: "shared definition after hidden use", body: "x <object>[hidden][r]</object> [visible][r] [again][r]\n\n[r]: /attachments/01A", kept: []int{0}},
		{name: "distinct visible definitions", body: "[a][one] [b][two]\n\n[one]: /attachments/01A\n[two]: /attachments/01A", kept: []int{0, 1}},
		{name: "all visible raw attributes", body: "<img src='/attachments/01A'> <a href=/attachments/01A>file</a>", kept: []int{0, 1}},
		{name: "HTML block lines and closure", body: "<div>\n<img src=\"/attachments/01A\">\n<img src=\"/attachments/01A\"></div>", kept: []int{0, 1}},
		{name: "multiline raw attributes", body: "<div>\n<img\n src=\"/attachments/01A\">\n<a\n href=\"/attachments/01A\">file</a></div>", kept: []int{0, 1}},
		{name: "nested visible image and link", body: "[![image](/attachments/01A)](/attachments/01A)", kept: []int{0, 1}},
		{name: "shared definition and direct reference", body: "[one][r] ![two][r] [three](/attachments/01A)\n\n[r]: /attachments/01A", kept: []int{0, 1}},
	}
	for _, tt := range tests {
		for _, newline := range []string{"\n", "\r\n"} {
			t.Run(tt.name+fmt.Sprintf("/%q", newline), func(t *testing.T) {
				t.Parallel()
				body := strings.ReplaceAll(tt.body, "\n", newline)
				destination := "/attachments/01A"
				var offsets []int
				for position := 0; position < len(body); {
					relative := strings.Index(body[position:], destination)
					if relative < 0 {
						break
					}
					offsets = append(offsets, position+relative)
					position += relative + len(destination)
				}
				matches := markup.ScanAttachmentRefMatches(body)
				if len(matches) != len(tt.kept) {
					t.Fatalf("matches=%+v, want destination indices %v", matches, tt.kept)
				}
				want := body
				for i := len(tt.kept) - 1; i >= 0; i-- {
					start := offsets[tt.kept[i]]
					match := matches[i]
					if match.Start != start || match.Stop != start+len(destination) || match.AttachmentID != "01A" {
						t.Errorf("match=%+v, want destination at %d", match, start)
					}
					want = want[:start] + "attachments/a.png" + want[start+len(destination):]
				}
				if got := RewriteAttachmentLinks(body, map[model.AttachmentID]string{"01A": "a.png"}); got != want {
					t.Errorf("rewrite=%q, want %q", got, want)
				}
				if ids := markup.ExtractAttachmentIDs(body); !slices.Equal(ids, []string{"01A"}) {
					t.Errorf("IDs=%v, want only 01A", ids)
				}
				rendered := markup.RenderMarkdown(body)
				if !strings.Contains(rendered, destination) {
					t.Fatalf("fixture must retain a visible reference: %s", rendered)
				}
				rewrittenHTML := markup.RenderMarkdown(want)
				if strings.Count(rewrittenHTML, "attachments/a.png") != strings.Count(rendered, destination) {
					t.Errorf("visible reference count changed: before=%q after=%q", rendered, rewrittenHTML)
				}
			})
		}
	}
}
