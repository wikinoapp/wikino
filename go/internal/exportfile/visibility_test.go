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
		{name: "object内の定義", body: "[visible][r]\n\nx <object>\n\n[r]: /attachments/01A\n\n</object>", live: true},
		{name: "textarea内の定義", body: "[visible][r]\n\nx <textarea>\n\n[r]: /attachments/01A\n\n</textarea>", live: true},
		{name: "object内だけの使用", body: "x <object>[hidden][r]</object>\n\n[r]: /attachments/01A"},
		{name: "textarea内だけの使用", body: "x <textarea>[hidden][r]</textarea>\n\n[r]: /attachments/01A"},
		{name: "表示される使用より前の隠れた使用", body: "x <object>[hidden][r]</object> [visible][r]\n\n[r]: /attachments/01A", live: true},
		{name: "隠れた使用より前の表示される使用", body: "[visible][r] <textarea>[hidden][r]</textarea>\n\n[r]: /attachments/01A", live: true},
		{name: "参照の画像が1つの定義を共有する", body: "![r] ![r][] ![image][r]\n\n[r]: /attachments/01A", live: true},
		{name: "参照のリンクが1つの定義を共有する", body: "[r] [r][] [file][r]\n\n[r]: /attachments/01A", live: true},
		{name: "正規化されたラベル", body: "[file][ STRASSE  label ]\n\n[Straße label]: /attachments/01A", live: true},
		{name: "最初の定義が優先される", body: "[file][r]\n\n[r]: /attachments/01A\n[r]: /attachments/01B", live: true},
		{name: "先の定義に隠れた添付ファイルの定義", body: "[file][r]\n\n[r]: /elsewhere\n[r]: /attachments/01A"},
		{name: "使われていない定義", body: "[r]: /attachments/01A"},
		{name: "code内は使用に数えない", body: "`[file][r]`\n\n[r]: /attachments/01A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rendered := markup.RenderMarkdown(tt.body)
			live := strings.Contains(rendered, `href="/attachments/01A"`) || strings.Contains(rendered, `src="/attachments/01A"`)
			if live != tt.live {
				t.Fatalf("描画された参照の表示 = %v、期待値 = %v: %s", live, tt.live, rendered)
			}
			var wantIDs []string
			want := tt.body
			if tt.live {
				wantIDs = []string{"01A"}
				want = strings.Replace(tt.body, "/attachments/01A", "attachments/a.pdf", 1)
			}
			if ids := markup.ExtractAttachmentIDs(tt.body); !slices.Equal(ids, wantIDs) {
				t.Errorf("ExtractAttachmentIDs() = %v、期待値 = %v", ids, wantIDs)
			}
			matches := markup.ScanAttachmentRefMatches(tt.body)
			if len(matches) != len(wantIDs) {
				t.Fatalf("matches = %+v、期待値 = %d", matches, len(wantIDs))
			}
			if tt.live {
				match := matches[0]
				if match.Start != strings.Index(tt.body, "/attachments/01A") ||
					match.Stop != match.Start+len("/attachments/01A") || match.InHTMLAttribute {
					t.Errorf("match = %+v、期待値 = 定義の参照先", match)
				}
			}
			names := map[model.AttachmentID]string{"01A": "a.pdf", "01B": "b.pdf"}
			if got := RewriteAttachmentLinks(tt.body, names); got != want {
				t.Errorf("RewriteAttachmentLinks() = %q、期待値 = %q", got, want)
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
					t.Fatalf("描画された画像が無い: %s", rendered)
				}
				converted := markup.ReplaceWikilinks(body, "T", "space", []markup.PageLocation{
					{Key: markup.WikilinkKey{Raw: "A"}, TopicName: "T", PageNumber: 1, PageTitle: "A"},
				})
				if !strings.Contains(converted, `href="/s/space/pages/1"`) {
					t.Fatalf("描画されたWikiリンクが無い: %s", converted)
				}
				wikiMatches := markup.ScanWikilinkMatches(body, "T")
				if len(wikiMatches) != 1 || wikiMatches[0].Key.PageTitle != "A" {
					t.Errorf("Wikiリンクの一致 = %+v、期待値 = A", wikiMatches)
				}
				attachmentMatches := markup.ScanAttachmentRefMatches(body)
				if len(attachmentMatches) != 1 || attachmentMatches[0].AttachmentID != "01B" {
					t.Errorf("添付ファイルの一致 = %+v、期待値 = 01B", attachmentMatches)
				}
				if got := RewriteWikilinks(body, "T", map[PageRef]string{{TopicName: "T", PageTitle: "A"}: "T/A.md"}); got != strings.Replace(body, "[[A]]", "[[T/A.md]]", 1) {
					t.Errorf("予期しないWikiリンクの書き換え: %q", got)
				}
				if got := RewriteAttachmentLinks(body, map[model.AttachmentID]string{"01B": "b.png"}); got != strings.Replace(body, "/attachments/01B", "attachments/b.png", 1) {
					t.Errorf("予期しない添付ファイルの書き換え: %q", got)
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
		{name: "表が暗黙の段落を閉じる", body: "<code><table><tr><td>[[A]]</td></tr></table></code>", want: []string{"A"}},
		{name: "表を閉じた後ろでcodeが戻る", body: "<code><table><tr><td>[[A]]</td></tr></table>[[B]]</code>", want: []string{"A"}},
		{name: "HTMLブロックは表の前後のcodeを保つ", body: "<div><code><table><tr><td>[[A]]</td></tr></table>[[B]]</code></div>"},
		{name: "preの中の表は保護される", body: "<pre><table><tr><td>[[A]]</td></tr></table>[[B]]</pre>"},
		{name: "表の外に出されたcodeの後ろでもセルは表示される", body: "<table><code><tr><td>[[A]]</td></tr></table>[[B]]", want: []string{"A"}},
		{name: "再構築されたcodeの中のMarkdownの表", body: "x <code>\n\n| [[A]] |\n| --- |\n\n[[B]]"},
		{name: "HTMLの終了タグがMarkdownの引用を閉じる", body: "> <pre>\n> </blockquote>[[A]]", want: []string{"A"}},
		{name: "HTMLの終了タグがMarkdownのリスト項目を閉じる", body: "- <pre>\n  </li>[[A]]", want: []string{"A"}},
		{name: "pre内の引用記号はテキスト", body: "> <pre>\n> > </blockquote>[[A]]", want: []string{"A"}},
		{name: "表と行の間の空白", body: "<code><table>\n  <tr><td>[[A]]</td></tr></table></code>", want: []string{"A"}},
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
			rendered := markup.ReplaceWikilinks(tt.body, "T", "space", locations)
			for i, title := range []string{"A", "B"} {
				if got := strings.Contains(rendered, fmt.Sprintf(`href="/s/space/pages/%d"`, i+1)); got != slices.Contains(tt.want, title) {
					t.Fatalf("%sの描画時の表示が想定と異なる: %s", title, rendered)
				}
			}
			var titles []string
			for _, match := range markup.ScanWikilinkMatches(tt.body, "T") {
				titles = append(titles, match.Key.PageTitle)
				if tt.body[match.Start:match.Stop] != "[["+match.Key.PageTitle+"]]" {
					t.Errorf("ソースの範囲が正しくない: %+v", match)
				}
			}
			if !slices.Equal(titles, tt.want) {
				t.Errorf("スキャンしたタイトル = %v、期待値 = %v", titles, tt.want)
			}
			want := tt.body
			for _, title := range tt.want {
				want = strings.ReplaceAll(want, "[["+title+"]]", "[[T/"+title+".md]]")
			}
			if got := RewriteWikilinks(tt.body, "T", paths); got != want {
				t.Errorf("RewriteWikilinks() = %q、期待値 = %q", got, want)
			}
		})
	}
}

func TestRewriteWikilinks_RendererDepthLimit(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("<code>", 600) + strings.Repeat("</code>", 600) + "[[A]]"
	rendered := markup.RenderMarkdown(body)
	converted := markup.ReplaceWikilinks(body, "T", "space", []markup.PageLocation{
		{Key: markup.WikilinkKey{Raw: "A"}, TopicName: "T", PageNumber: 1, PageTitle: "A"},
	})
	if converted != rendered {
		t.Fatalf("表示側がパーサーの深さの上限を超えた本文を変換した")
	}
	if matches := markup.ScanWikilinkMatches(body, "T"); len(matches) != 0 {
		t.Errorf("matches = %+v、期待値 = 表示側の深さの上限を超えた変換が無い", matches)
	}
	if got := RewriteWikilinks(body, "T", map[PageRef]string{{TopicName: "T", PageTitle: "A"}: "T/A.md"}); got != body {
		t.Errorf("表示側で変換できない本文をエクスポートが変更した: %q", got)
	}
}

func TestRewriteWikilinks_CodeAfterRawTextElement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "コードスパン", body: "`[[A]]`"},
		{name: "フェンスで囲んだブロック", body: "```\n[[A]]\n```"},
		{name: "インデントしたブロック", body: "    [[A]]\n"},
		{name: "フェンスの言語指定", body: "```[[A]]\nx\n```"},
		{name: "言語指定の後ろのフェンス", body: "```x [[A]]\ny\n```"},
		{name: "xmpの後ろのコードスパン", body: "<xmp>`[[A]]`", want: []string{"A"}},
		{name: "xmpの後ろのフェンスで囲んだブロック", body: "<xmp>\n\n```\n[[A]]\n```", want: []string{"A"}},
		{name: "xmpの後ろのインデントしたブロック", body: "<xmp>\n\n    [[A]]\n", want: []string{"A"}},
		{name: "xmpの後ろのフェンスの言語指定", body: "<xmp>\n\n```[[A]]\nx\n```", want: []string{"A"}},
		{name: "xmpの後ろでも言語指定の後ろのテキストは除く", body: "<xmp>\n\n```x [[A]]\ny\n```"},
		{name: "plaintextの後ろのコードスパン", body: "<plaintext>`[[A]]`", want: []string{"A"}},
		{name: "textareaの後ろのコードスパン", body: "<textarea>x\n\n`[[A]]`", want: []string{"A"}},
		{name: "閉じたxmpの後ろでcodeが戻る", body: "<xmp>y</xmp>\n\n`[[A]]`"},
		{name: "コードスパンがテキストを分断する", body: "[[ と書く `x` [[A]]", want: []string{"A"}},
		{name: "コードスパンが対の括弧を分断する", body: "[[A`x`]]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertWikilinkParity(t, tt.body, tt.want)
		})
	}
}

// assertWikilinkParityは、bodyのどのWikiリンクが読み手に届くかについて、表示・走査・
// 書き換えが一致することを確かめる。wantは画面がリンクに変える (べき) ページタイトル。
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

	rendered := markup.ReplaceWikilinks(body, "T", "space", locations)
	for i, title := range titles {
		got := strings.Contains(rendered, fmt.Sprintf(`href="/s/space/pages/%d"`, i+1))
		if got != slices.Contains(want, title) {
			t.Fatalf("%sの描画時の表示 = %v、期待値 = %v: %s", title, got, !got, rendered)
		}
	}

	var scanned []string
	for _, match := range markup.ScanWikilinkMatches(body, "T") {
		scanned = append(scanned, match.Key.PageTitle)
		if body[match.Start:match.Stop] != "[["+match.Key.PageTitle+"]]" {
			t.Errorf("ソースの範囲が正しくない: %+v", match)
		}
	}
	if !slices.Equal(scanned, want) {
		t.Errorf("スキャンしたタイトル = %v、期待値 = %v", scanned, want)
	}

	rewritten := body
	for _, title := range want {
		rewritten = strings.ReplaceAll(rewritten, "[["+title+"]]", "[[T/"+title+".md]]")
	}
	if got := RewriteWikilinks(body, "T", paths); got != rewritten {
		t.Errorf("RewriteWikilinks() = %q、期待値 = %q", got, rewritten)
	}
}

func TestRewriteAttachmentLinks_DroppedBySanitizer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		live bool
	}{
		{name: "引用内の閉じていないコメント", body: "><!--\n\n<img src=\"/attachments/01A\">"},
		{name: "セル内の閉じていないコメント", body: "<td><!--\n\n<img src=\"/attachments/01A\">"},
		{name: "Markdownの画像の前の閉じていないコメント", body: "><!--\n\n![x](/attachments/01A)"},
		{name: "リンクの前の閉じていないタグ", body: "<td\n\n<a href=\"/attachments/01A\">"},
		{name: "閉じたコメントの後ろの画像は有効", body: "<!-- y -->\n\n<img src=\"/attachments/01A\">", live: true},
		{name: "閉じたコメントを含む引用", body: "> <!-- y -->\n\n<img src=\"/attachments/01A\">", live: true},
		{name: "HTMLのcode要素の中の画像", body: "<code><img src=\"/attachments/01A\"></code>", live: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rendered := markup.RenderMarkdown(tt.body)
			live := strings.Contains(rendered, `href="/attachments/01A"`) ||
				strings.Contains(rendered, `src="/attachments/01A"`)
			if live != tt.live {
				t.Fatalf("描画された参照の表示 = %v、期待値 = %v: %s", live, tt.live, rendered)
			}

			var wantIDs []string
			want := tt.body
			if tt.live {
				wantIDs = []string{"01A"}
				want = strings.Replace(tt.body, "/attachments/01A", "attachments/a.png", 1)
			}
			if ids := markup.ExtractAttachmentIDs(tt.body); !slices.Equal(ids, wantIDs) {
				t.Errorf("ExtractAttachmentIDs() = %v、期待値 = %v", ids, wantIDs)
			}
			if got := RewriteAttachmentLinks(tt.body, map[model.AttachmentID]string{"01A": "a.png"}); got != want {
				t.Errorf("RewriteAttachmentLinks() = %q、期待値 = %q", got, want)
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
		{name: "閉じていないコメント", body: "<!--[[A]]"},
		{name: "閉じたコメント", body: "<!--[[A]]-->"},
		{name: "インラインのコメント", body: "x <!-- [[A]] -->"},
		{name: "doctype", body: "x\n<!DOCTYPE [[A]]>"},
		{name: "タグの属性", body: `<div title="[[A]]">y</div>`},
		{name: "閉じていないpreの中のコメント", body: "b<pre><textarea>\n<!--[[A]]"},
		{name: "xmpはコメントをテキストにする", body: "<xmp>\n\n<!--[[A]]", want: []string{"A"}},
		{name: "textareaはコメントをテキストにする", body: "<textarea>\n<!--[[A]]", want: []string{"A"}},
		{name: "xmpはpreもテキストにする", body: "<xmp>\n\n<pre>\n\n<!--[[A]]", want: []string{"A"}},
		{name: "xmpはdoctypeをテキストにする", body: "<xmp>\n\n<!DOCTYPE [[A]]>", want: []string{"A"}},
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
		{name: "隠れたHTMLの画像", body: "![visible](/attachments/01A)\n\n> <!--\n\n<img src=\"/attachments/01A\">", kept: []int{0}},
		{name: "隠れたMarkdownの画像", body: "![visible](/attachments/01A)\n\n> <!--\n\n![hidden](/attachments/01A)", kept: []int{0}},
		{name: "隠れたHTMLのアンカー", body: "[visible](/attachments/01A)\n\n> <!--\n\n<a href=\"/attachments/01A\">hidden</a>", kept: []int{0}},
		{name: "隠れたMarkdownのアンカー", body: "[visible](/attachments/01A)\n\n> <!--\n\n[hidden](/attachments/01A)", kept: []int{0}},
		{name: "隠れた参照定義", body: "[visible](/attachments/01A)\n\n> <!--\n\n[hidden][r]\n\n[r]: /attachments/01A", kept: []int{0}},
		{name: "表示される使用を持つ共有された定義", body: "[visible][r]\n\n> <!--\n\n[hidden][r]\n\n[r]: /attachments/01A", kept: []int{0}},
		{name: "隠れた使用の後ろの共有された定義", body: "x <object>[hidden][r]</object> [visible][r] [again][r]\n\n[r]: /attachments/01A", kept: []int{0}},
		{name: "表示される別々の定義", body: "[a][one] [b][two]\n\n[one]: /attachments/01A\n[two]: /attachments/01A", kept: []int{0, 1}},
		{name: "すべて表示されるHTMLの属性", body: "<img src='/attachments/01A'> <a href=/attachments/01A>file</a>", kept: []int{0, 1}},
		{name: "HTMLブロックの行と閉じタグ", body: "<div>\n<img src=\"/attachments/01A\">\n<img src=\"/attachments/01A\"></div>", kept: []int{0, 1}},
		{name: "複数行のHTMLの属性", body: "<div>\n<img\n src=\"/attachments/01A\">\n<a\n href=\"/attachments/01A\">file</a></div>", kept: []int{0, 1}},
		{name: "入れ子になった表示される画像とリンク", body: "[![image](/attachments/01A)](/attachments/01A)", kept: []int{0, 1}},
		{name: "共有された定義と直接の参照", body: "[one][r] ![two][r] [three](/attachments/01A)\n\n[r]: /attachments/01A", kept: []int{0, 1}},
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
					t.Fatalf("matches = %+v、期待値 = 参照先の位置%v", matches, tt.kept)
				}
				want := body
				for i := len(tt.kept) - 1; i >= 0; i-- {
					start := offsets[tt.kept[i]]
					match := matches[i]
					if match.Start != start || match.Stop != start+len(destination) || match.AttachmentID != "01A" {
						t.Errorf("match = %+v、期待値 = 位置%dの参照先", match, start)
					}
					want = want[:start] + "attachments/a.png" + want[start+len(destination):]
				}
				if got := RewriteAttachmentLinks(body, map[model.AttachmentID]string{"01A": "a.png"}); got != want {
					t.Errorf("rewrite = %q、期待値 = %q", got, want)
				}
				if ids := markup.ExtractAttachmentIDs(body); !slices.Equal(ids, []string{"01A"}) {
					t.Errorf("IDs = %v、期待値 = 01Aのみ", ids)
				}
				rendered := markup.RenderMarkdown(body)
				if !strings.Contains(rendered, destination) {
					t.Fatalf("フィクスチャに表示される参照が残っていない: %s", rendered)
				}
				rewrittenHTML := markup.RenderMarkdown(want)
				if strings.Count(rewrittenHTML, "attachments/a.png") != strings.Count(rendered, destination) {
					t.Errorf("表示される参照の件数が変わった: 変更前 = %q、変更後 = %q", rendered, rewrittenHTML)
				}
			})
		}
	}
}
