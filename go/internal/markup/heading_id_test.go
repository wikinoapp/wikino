package markup

import (
	"regexp"
	"slices"
	"testing"
)

// headingIDAttrRegexは、描画結果の見出し要素に付いたidを取り出す。
var headingIDAttrRegex = regexp.MustCompile(`<h[1-6] id="([^"]*)"`)

// TestRenderMarkdown_HeadingIDsは、見出しのidがRails版 (commonmarker 2.3.2 / comrak 0.41) と
// 同じ規則で付き、サニタイズを経ても残ることを固定する。期待値はRails版と同じオプションで
// commonmarker 2.3.2に同じ本文を渡したときの出力である。
func TestRenderMarkdown_HeadingIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{name: "日本語の見出し", body: "## 見出し", want: []string{"見出し"}},
		{name: "英字は小文字になり、半角スペースは-になる", body: "## Wiki リンク", want: []string{"wiki-リンク"}},
		{
			name: "重複した見出しには連番が付く",
			body: "## 見出し\n\n## 見出し\n\n## 見出し",
			want: []string{"見出し", "見出し-1", "見出し-2"},
		},
		{
			name: "連番は既存のidと重ならない番号になる",
			body: "## a-1\n\n## a\n\n## a",
			want: []string{"a-1", "a", "a-2"},
		},
		{
			// comrakは空のidを出力するが、何も指せないため付けない。連番は空のidも数えてcomrakに揃える。
			name: "記号だけの見出しは空のidになり、2つ目は-1になる",
			body: "## !!!\n\n## ???",
			want: []string{"-1"},
		},
		{
			name: "インラインコードや強調は記法を除いたテキストになる",
			body: "## `code` と **太字** と *斜体*",
			want: []string{"code-と-太字-と-斜体"},
		},
		{name: "記号は取り除かれる", body: "## C++ & C#: 比較", want: []string{"c--c-比較"}},
		{
			name: "エスケープと文字参照は解決した後の文字で数える",
			body: `## \*エスケープ\* &amp; &#12354;`,
			want: []string{"エスケープ--あ"},
		},
		{
			name: "インラインコードの中身はエスケープを解決しない",
			body: "## `a\\*b` &lt;x&gt;",
			want: []string{"ab-x"},
		},
		{name: "インラインのHTMLタグは数えない", body: "## <span>タグ</span>の中", want: []string{"タグの中"}},
		{
			name: "リンクのテキストと画像の代替テキストを数える",
			body: "## [リンク](https://example.com) と ![画像](/a.png)",
			want: []string{"リンク-と-画像"},
		},
		{name: "自動リンクはURLを数える", body: "## <https://example.com/a_b>", want: []string{"httpsexamplecoma_b"}},
		{
			name: "自動リンクの文字参照は解決した後の文字で数える",
			body: "## <https://example.com/?a=1&amp;b=2>",
			want: []string{"httpsexamplecoma1b2"},
		},
		{name: "URLだけの自動リンクもURLを数える", body: "## https://example.com/x", want: []string{"httpsexamplecomx"}},
		{name: "setext見出しの改行は半角スペースとして数える", body: "複数行の\nsetext見出し\n===", want: []string{"複数行の-setext見出し"}},
		{
			name: "Wikiリンクは記法の記号を除いたテキストになる",
			body: "## [[Markdown]] と [[トピック/ページ]]",
			want: []string{"markdown-と-トピックページ"},
		},
		{name: "全角スペースとタブは取り除かれる", body: "## 全角　スペース\tタブ", want: []string{"全角スペースタブ"}},
		{name: "取り消し線の中のテキストを数える", body: "## ~~取り消し~~", want: []string{"取り消し"}},
		{name: "アクセント付きの英字とアンダースコアは残る", body: "## snake_case と ÀÉÎ", want: []string{"snake_case-と-àéî"}},
		{name: "数字だけでなく丸数字も残り、絵文字は取り除かれる", body: "## 😀 絵文字 ①②", want: []string{"-絵文字-①②"}},
		{
			name: "引用やリストの中の見出しにも付く",
			body: "> ## 引用の中\n\n- ## リストの中",
			want: []string{"引用の中", "リストの中"},
		},
		{name: "本文に直接書いた見出しには付けない", body: "<h2>生の見出し</h2>\n\n## 生の見出し", want: []string{"生の見出し"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got []string
			for _, m := range headingIDAttrRegex.FindAllStringSubmatch(RenderMarkdown(tt.body), -1) {
				got = append(got, m[1])
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("見出しのid = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

// TestRenderMarkdown_PageAnchorReachesJapaneseHeadingは、日本語の見出しを指すページ内リンクが
// 見出しのidと一致することを固定する。
func TestRenderMarkdown_PageAnchorReachesJapaneseHeading(t *testing.T) {
	t.Parallel()

	got := RenderMarkdown("## 見出し\n\n[見出しへジャンプ](#見出し)")

	for _, want := range []string{`<h2 id="見出し">見出し</h2>`, `href="#%E8%A6%8B%E5%87%BA%E3%81%97"`} {
		if !regexp.MustCompile(regexp.QuoteMeta(want)).MatchString(got) {
			t.Errorf("RenderMarkdown() = %q、%q を含むことを期待", got, want)
		}
	}
}
