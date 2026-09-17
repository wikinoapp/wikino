package markup

import (
	"strconv"
	"testing"
)

func TestPlainText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bodyHTML string
		want     string
	}{
		{
			name:     "空文字列",
			bodyHTML: "",
			want:     "",
		},
		{
			name:     "単一の段落からテキストを取り出す",
			bodyHTML: "<p>本文のテキスト</p>",
			want:     "本文のテキスト",
		},
		{
			name:     "隣接する段落はスペースで区切る",
			bodyHTML: "<p>1 つ目の段落</p><p>2 つ目の段落</p>",
			want:     "1 つ目の段落 2 つ目の段落",
		},
		{
			name:     "セクション要素の境界はスペースで区切る",
			bodyHTML: "<article>記事</article><aside>補足</aside><section>節</section>",
			want:     "記事 補足 節",
		},
		{
			name:     "detailsとsummaryの境界はスペースで区切る",
			bodyHTML: "<details><summary>概要</summary>詳細</details><hgroup>見出しグループ</hgroup>",
			want:     "概要 詳細 見出しグループ",
		},
		{
			name:     "定義リストの項目はスペースで区切る",
			bodyHTML: "<dl><dt>用語</dt><dd>定義</dd></dl>",
			want:     "用語 定義",
		},
		{
			name:     "figureとcaptionの境界はスペースで区切る",
			bodyHTML: "<figure>図<figcaption>説明</figcaption></figure><figure>別の図</figure>",
			want:     "図 説明 別の図",
		},
		{
			name:     "テーブルのcaptionと各セクションをスペースで区切る",
			bodyHTML: "<table><caption>表題</caption><thead><tr><th>見出し</th></tr></thead><tbody><tr><td>本文</td></tr></tbody><tfoot><tr><td>脚注</td></tr></tfoot></table>",
			want:     "表題 見出し 本文 脚注",
		},
		{
			name:     "インライン要素は前後にスペースを入れない",
			bodyHTML: "<p>これは<strong>太字</strong>です</p>",
			want:     "これは太字です",
		},
		{
			name:     "リンクのテキストは残す",
			bodyHTML: `<p>詳しくは<a href="/s/foo/pages/1">こちら</a></p>`,
			want:     "詳しくはこちら",
		},
		{
			name:     "リスト項目はスペースで区切る",
			bodyHTML: "<ul><li>りんご</li><li>みかん</li></ul>",
			want:     "りんご みかん",
		},
		{
			name:     "brは区切りとして扱う",
			bodyHTML: "<p>1 行目<br>2 行目</p>",
			want:     "1 行目 2 行目",
		},
		{
			name:     "見出しと本文を区切る",
			bodyHTML: "<h2>見出し</h2><p>本文</p>",
			want:     "見出し 本文",
		},
		{
			name:     "HTMLエンティティをデコードする",
			bodyHTML: "<p>A &amp; B &lt;C&gt;</p>",
			want:     "A & B <C>",
		},
		{
			name:     "連続する空白と改行を1個のスペースにまとめる",
			bodyHTML: "<p>語   と\n\n語</p>",
			want:     "語 と 語",
		},
		{
			name:     "scriptの内容は取り出さない",
			bodyHTML: "<p>本文</p><script>alert(1)</script>",
			want:     "本文",
		},
		{
			name:     "styleの内容は取り出さない",
			bodyHTML: "<style>p { color: red }</style><p>本文</p>",
			want:     "本文",
		},
		{
			name:     "画像だけの本文は空文字列になる",
			bodyHTML: `<p><img src="/attachments/1" alt="図"></p>`,
			want:     "",
		},
		{
			name:     "コードブロックのテキストは残す",
			bodyHTML: "<pre><code>go build ./...</code></pre>",
			want:     "go build ./...",
		},
		{
			name:     "テーブルのセルをスペースで区切る",
			bodyHTML: "<table><tr><td>左</td><td>右</td></tr></table>",
			want:     "左 右",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := PlainText(tt.bodyHTML, 0); got != tt.want {
				t.Errorf("PlainText() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

// TestPlainText_MaxRunesは上限がrune単位であることを固定する。ブロックの境界が1文字ぶんを
// 占めること、0以下が上限無しを意味することも含む。
func TestPlainText_MaxRunes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bodyHTML string
		maxRunes int
		want     string
	}{
		{
			name:     "上限までで打ち切る",
			bodyHTML: "<p>abc</p><p>def</p>",
			maxRunes: 3,
			want:     "abc",
		},
		{
			name:     "ブロックの境界も1文字として数える",
			bodyHTML: "<p>abc</p><p>def</p>",
			maxRunes: 4,
			want:     "abc ",
		},
		{
			name:     "境界の次のブロックへ入る",
			bodyHTML: "<p>abc</p><p>def</p>",
			maxRunes: 5,
			want:     "abc d",
		},
		{
			name:     "上限が本文と同じ長さなら全文を返す",
			bodyHTML: "<p>abc</p><p>def</p>",
			maxRunes: 7,
			want:     "abc def",
		},
		{
			name:     "上限が本文より長ければ全文を返す",
			bodyHTML: "<p>abc</p><p>def</p>",
			maxRunes: 100,
			want:     "abc def",
		},
		{
			name:     "0は上限無し",
			bodyHTML: "<p>abc</p><p>def</p>",
			maxRunes: 0,
			want:     "abc def",
		},
		{
			name:     "負の値は上限無し",
			bodyHTML: "<p>abc</p><p>def</p>",
			maxRunes: -1,
			want:     "abc def",
		},
		{
			name:     "バイト数ではなく文字数で数える",
			bodyHTML: "<p>あいうえお</p>",
			maxRunes: 3,
			want:     "あいう",
		},
		{
			name:     "先頭の空白は上限を消費しない",
			bodyHTML: "<p>   abc</p>",
			maxRunes: 2,
			want:     "ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := PlainText(tt.bodyHTML, tt.maxRunes); got != tt.want {
				t.Errorf("PlainText() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

// TestPlainText_MaxRunesReturnsPrefixは、上限付きの結果が上限無しの結果の先頭maxRunes文字
// そのものになることを固定する。PageForShow.MetaDescriptionはこれに依存しており、上限より1文字
// 多く要求して長さだけで切り詰めの要否を決めている。
func TestPlainText_MaxRunesReturnsPrefix(t *testing.T) {
	t.Parallel()

	bodies := []string{
		"<h2>見出し</h2><p>これは<strong>太字</strong>です</p><ul><li>りんご</li><li>みかん</li></ul>",
		"<p>1 行目<br>2 行目</p><table><tr><td>左</td><td>右</td></tr></table>",
		"<section>First</section><dl><dt>Term</dt><dd>Definition</dd></dl>",
	}

	for i, bodyHTML := range bodies {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			full := []rune(PlainText(bodyHTML, 0))
			for maxRunes := 1; maxRunes <= len(full)+2; maxRunes++ {
				want := string(full[:min(maxRunes, len(full))])
				if got := PlainText(bodyHTML, maxRunes); got != want {
					t.Errorf("PlainText(_, %d) = %q、期待値 = %q", maxRunes, got, want)
				}
			}
		})
	}
}
