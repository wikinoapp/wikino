package markup

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// wikilinkTestLocations resolves ページ1 and ページ2 of topic トピックA to pages 1 and 2.
//
// [Ja] wikilinkTestLocations はトピックA のページ1・ページ2 をページ番号 1・2 に解決する。
func wikilinkTestLocations() []PageLocation {
	var locations []PageLocation
	for i, title := range []string{"ページ1", "ページ2"} {
		locations = append(locations, PageLocation{
			Key:        WikilinkKey{Raw: title, TopicName: "トピックA", PageTitle: title},
			TopicName:  "トピックA",
			PageID:     model.PageID(fmt.Sprintf("page-id-%d", i+1)),
			PageNumber: i + 1,
			PageTitle:  title,
		})
	}

	return locations
}

// assertDisplayMatchesScan checks that the screen links exactly the wiki links ScanWikilinkMatches
// reports for body, and returns the rendered HTML. want holds the page titles that reach the reader
// as links, in order.
//
// [Ja] assertDisplayMatchesScan は、画面が ScanWikilinkMatches の報告する Wiki リンクとちょうど
// 同じものをリンクにすることを確かめ、レンダリング結果の HTML を返す。want は読み手にリンクとして
// 届くページタイトルの並び。
func assertDisplayMatchesScan(t *testing.T, body string, want []string) string {
	t.Helper()

	locations := wikilinkTestLocations()
	rendered := ReplaceWikilinks(body, "トピックA", "my-space", locations)
	for _, location := range locations {
		got := strings.Contains(rendered, fmt.Sprintf(`href="/s/my-space/pages/%d"`, location.PageNumber))
		if got != slices.Contains(want, location.PageTitle) {
			t.Errorf("rendered link for %s = %v, want %v: %s", location.PageTitle, got, !got, rendered)
		}
	}

	var scanned []string
	for _, match := range ScanWikilinkMatches(body, "トピックA") {
		if findPageLocation(match.Key, locations) != nil {
			scanned = append(scanned, match.Key.PageTitle)
		}
	}
	if !slices.Equal(scanned, want) {
		t.Errorf("scanned resolvable titles = %v, want %v", scanned, want)
	}

	return rendered
}

func TestReplaceWikilinks_ExistingPage(t *testing.T) {
	t.Parallel()

	got := ReplaceWikilinks("テキスト [[ページ1]] テキスト", "トピックA", "my-space", wikilinkTestLocations())
	want := "<p>テキスト <a href=\"/s/my-space/pages/1\">ページ1</a> テキスト</p>\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_NonExistingPageStaysAsWritten(t *testing.T) {
	t.Parallel()

	body := "テキスト [[存在しないページ]] テキスト"
	got := ReplaceWikilinks(body, "トピックA", "my-space", wikilinkTestLocations())
	if want := RenderMarkdown(body); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_TopicAndPageName(t *testing.T) {
	t.Parallel()

	locations := []PageLocation{{
		Key:        WikilinkKey{Raw: "トピックB/ページ2", TopicName: "トピックB", PageTitle: "ページ2"},
		TopicName:  "トピックB",
		PageID:     model.PageID("page-id-2"),
		PageNumber: 7,
		PageTitle:  "ページ2",
	}}
	got := ReplaceWikilinks("[[トピックB/ページ2]]", "トピックA", "my-space", locations)
	want := "<p><a href=\"/s/my-space/pages/7\">ページ2</a></p>\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_MixedResolvedAndUnresolved(t *testing.T) {
	t.Parallel()

	got := ReplaceWikilinks("> [[ページ1]] と [[存在しない]] と [[ページ2]]", "トピックA", "my-space", wikilinkTestLocations())
	want := "<blockquote>\n<p><a href=\"/s/my-space/pages/1\">ページ1</a> と [[存在しない]] と <a href=\"/s/my-space/pages/2\">ページ2</a></p>\n</blockquote>\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_NoWikilinks(t *testing.T) {
	t.Parallel()

	body := "Wikiリンクなしの **テキスト**"
	got := ReplaceWikilinks(body, "トピックA", "my-space", nil)
	if want := RenderMarkdown(body); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_EmptyInput(t *testing.T) {
	t.Parallel()

	if got := ReplaceWikilinks("", "トピックA", "my-space", nil); got != "" {
		t.Errorf("got: %q, want empty string", got)
	}
}

func TestReplaceWikilinks_LinkTextAndSpaceIdentifierAreEscaped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		body            string
		title           string
		spaceIdentifier model.SpaceIdentifier
		want            string
	}{
		{
			name: "ページタイトルの HTML 特殊文字", body: "[[ページ1]]", title: "ページ<script>", spaceIdentifier: "my-space",
			want: "<p><a href=\"/s/my-space/pages/1\">ページ&lt;script&gt;</a></p>\n",
		},
		{
			name: "アンパサンドを含むタイトル", body: "[[A & B]]", title: "A & B", spaceIdentifier: "my-space",
			want: "<p><a href=\"/s/my-space/pages/1\">A &amp; B</a></p>\n",
		},
		{
			name: "矢印を含むタイトル", body: "[[A -> B]]", title: "A -> B", spaceIdentifier: "my-space",
			want: "<p><a href=\"/s/my-space/pages/1\">A -&gt; B</a></p>\n",
		},
		{
			name: "引用符を含むタイトル", body: `[[A "B"]]`, title: `A "B"`, spaceIdentifier: "my-space",
			want: "<p><a href=\"/s/my-space/pages/1\">A &#34;B&#34;</a></p>\n",
		},
		{
			name: "括弧を含むタイトル", body: "[[日記 (2025)]]", title: "日記 (2025)", spaceIdentifier: "my-space",
			want: "<p><a href=\"/s/my-space/pages/1\">日記 (2025)</a></p>\n",
		},
		{
			name: "スペース識別子のエスケープ", body: "[[ページ1]]", title: "ページ1", spaceIdentifier: "space&id",
			want: "<p><a href=\"/s/space&amp;id/pages/1\">ページ1</a></p>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw := strings.TrimSuffix(strings.TrimPrefix(tt.body, "[["), "]]")
			locations := []PageLocation{{
				Key:        WikilinkKey{Raw: raw, TopicName: "トピックA", PageTitle: raw},
				TopicName:  "トピックA",
				PageID:     model.PageID("page-id-1"),
				PageNumber: 1,
				PageTitle:  tt.title,
			}}
			if got := ReplaceWikilinks(tt.body, "トピックA", tt.spaceIdentifier, locations); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestReplaceWikilinks_EscapedAndReferencedBracketsAreNotLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "バックスラッシュでエスケープした角括弧", body: `\[\[ページ1]]`},
		{name: "文字参照で書いた角括弧", body: "&#91;&#91;ページ1]]"},
		{name: "エスケープされたバックスラッシュの後ろは記法", body: `\\[[ページ1]]`, want: []string{"ページ1"}},
		{name: "HTML ブロックの中ではバックスラッシュは文字", body: "<div>\n\\[[ページ1]]\n</div>", want: []string{"ページ1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rendered := assertDisplayMatchesScan(t, tt.body, tt.want)
			if len(tt.want) == 0 && !strings.Contains(rendered, "[[ページ1]]") {
				t.Errorf("the notation should stay visible as text: %s", rendered)
			}
		})
	}
}

func TestReplaceWikilinks_DroppedEndTagDoesNotJoinText(t *testing.T) {
	t.Parallel()

	got := assertDisplayMatchesScan(t, "[</h1>[[ページ1]]", []string{"ページ1"})
	want := "<p>[<a href=\"/s/my-space/pages/1\">ページ1</a></p>\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_InlineElementsDoNotSplitText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "強調が対応の取れていない [[ に飲み込まれる", body: "[[ と書く *強調* [[ページ1]]"},
		{name: "打ち消し線も同じ", body: "[[ と書く ~~打ち消し~~ [[ページ1]]"},
		{name: "コードスパンはテキストを分ける", body: "[[ と書く `x` [[ページ1]]", want: []string{"ページ1"}},
		{name: "強調の中のリンク", body: "*強調 [[ページ1]] の中*", want: []string{"ページ1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertDisplayMatchesScan(t, tt.body, tt.want)
		})
	}
}

func TestReplaceWikilinks_LinkAcrossEmphasisDelimiter(t *testing.T) {
	t.Parallel()

	locations := []PageLocation{{
		Key:        WikilinkKey{Raw: "b* c", TopicName: "トピックA", PageTitle: "b* c"},
		TopicName:  "トピックA",
		PageID:     model.PageID("page-id-3"),
		PageNumber: 3,
		PageTitle:  "b* c",
	}}
	got := ReplaceWikilinks("*a [[b* c]] d*", "トピックA", "my-space", locations)
	want := "<p><em>a <a href=\"/s/my-space/pages/3\">b* c</a></em> d*</p>\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceWikilinks_SkippedAndDroppedElements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "コードスパン", body: "`[[ページ1]]` と [[ページ2]]", want: []string{"ページ2"}},
		{name: "コードブロック", body: "```\n[[ページ1]]\n```\n\n[[ページ2]]", want: []string{"ページ2"}},
		{name: "raw HTML の code", body: "<code>[[ページ1]]</code> [[ページ2]]", want: []string{"ページ2"}},
		{name: "raw HTML の pre", body: "<pre>[[ページ1]]</pre>\n\n[[ページ2]]", want: []string{"ページ2"}},
		{name: "raw HTML の a", body: `<a href="/x">[[ページ1]]</a> [[ページ2]]`, want: []string{"ページ2"}},
		{name: "raw HTML の script", body: "<script>[[ページ1]]</script>\n\n[[ページ2]]", want: []string{"ページ2"}},
		{name: "raw HTML の style", body: "<style>[[ページ1]]</style>\n\n[[ページ2]]", want: []string{"ページ2"}},
		{name: "サニタイズが中身ごと落とす iframe", body: "<iframe>[[ページ1]]</iframe>[[ページ2]]", want: []string{"ページ2"}},
		{name: "サニタイズが中身ごと落とす object", body: "x <object>[[ページ1]]</object> [[ページ2]]", want: []string{"ページ2"}},
		{name: "Markdown リンクのラベルの中", body: "[x [[ページ1]]](/x) [[ページ2]]", want: []string{"ページ2"}},
		{name: "サニタイズが要素を落とすリンクのラベルは見える", body: "[[[ページ1]]](javascript:alert(1)) [[ページ2]]", want: []string{"ページ1", "ページ2"}},
		{name: "raw text 要素の後ろではコードスパンがテキストになる", body: "<xmp>`[[ページ1]]` [[ページ2]]", want: []string{"ページ1", "ページ2"}},
		{name: "raw text 要素の後ろではコードブロックがテキストになる", body: "<xmp>\n\n```\n[[ページ1]]\n```\n\n[[ページ2]]", want: []string{"ページ1", "ページ2"}},
		{name: "raw text 要素の後ろではフェンスの言語がテキストになる", body: "<xmp>\n\n```[[ページ1]]\nx\n```\n\n[[ページ2]]", want: []string{"ページ1", "ページ2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rendered := assertDisplayMatchesScan(t, tt.body, tt.want)
			if strings.Contains(rendered, "Wikino") {
				t.Errorf("a marker leaked into the rendered HTML: %s", rendered)
			}
		})
	}
}

func TestReplaceWikilinks_BlockContexts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "HTML ブロックの行",
			body: "<div>\n[[ページ1]]\n</div>",
			want: "<div>\n<a href=\"/s/my-space/pages/1\">ページ1</a>\n</div>",
		},
		{
			name: "raw text 要素の中はテキストとして届く",
			body: "<textarea>\n[[ページ1]]\n</textarea>",
			want: "\n<a href=\"/s/my-space/pages/1\">ページ1</a>\n",
		},
		{
			name: "タブでインデントしたリスト項目",
			body: "- [[ページ1]]\n-\t[[ページ2]]",
			want: "<ul>\n<li><a href=\"/s/my-space/pages/1\">ページ1</a></li>\n<li><a href=\"/s/my-space/pages/2\">ページ2</a></li>\n</ul>\n",
		},
		{
			name: "表のセル",
			body: "| [[ページ1]] |\n| --- |\n| [[ページ2]] |",
			want: "<table>\n<thead>\n<tr>\n<th><a href=\"/s/my-space/pages/1\">ページ1</a></th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td><a href=\"/s/my-space/pages/2\">ページ2</a></td>\n</tr>\n</tbody>\n</table>\n",
		},
		{
			name: "見出し",
			body: "# [[ページ1]]",
			want: "<h1 id=\"1\"><a href=\"/s/my-space/pages/1\">ページ1</a></h1>\n",
		},
		{
			name: "行末の改行を保つ",
			body: "[[ページ1]]\n次の行",
			want: "<p><a href=\"/s/my-space/pages/1\">ページ1</a><br/>\n次の行</p>\n",
		},
		{
			name: "隣接するリンク",
			body: "[[ページ1]][[ページ2]]",
			want: "<p><a href=\"/s/my-space/pages/1\">ページ1</a><a href=\"/s/my-space/pages/2\">ページ2</a></p>\n",
		},
		{
			name: "CRLF の本文",
			body: "[[ページ1]]\r\n\r\n[[ページ2]]",
			want: "<p><a href=\"/s/my-space/pages/1\">ページ1</a></p>\n<p><a href=\"/s/my-space/pages/2\">ページ2</a></p>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ReplaceWikilinks(tt.body, "トピックA", "my-space", wikilinkTestLocations()); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
