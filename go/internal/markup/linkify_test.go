package markup

import (
	"strings"
	"testing"
)

func TestLinkifyPlainText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		text         string
		wantContains []string
		wantExcludes []string
	}{
		{
			name: "通常のテキストがそのまま表示される",
			text: "Hello, World",
			wantContains: []string{
				"Hello, World",
			},
		},
		{
			name: "HTMLタグがエスケープされる",
			text: "<script>alert('xss')</script>",
			wantContains: []string{
				"&lt;script&gt;",
				"&lt;/script&gt;",
				"&#39;xss&#39;",
			},
			wantExcludes: []string{
				"<script>",
				"</script>",
			},
		},
		{
			name: "https URL がリンク化される",
			text: "https://example.com",
			wantContains: []string{
				`<a href="https://example.com"`,
				`rel="noopener noreferrer"`,
				`target="_blank"`,
				`>https://example.com</a>`,
			},
		},
		{
			name: "http URL がリンク化される",
			text: "http://example.com",
			wantContains: []string{
				`<a href="http://example.com"`,
				`>http://example.com</a>`,
			},
		},
		{
			name: "テキスト中の URL がリンク化される",
			text: "Visit https://example.com for more info",
			wantContains: []string{
				"Visit ",
				`<a href="https://example.com"`,
				` for more info`,
			},
		},
		{
			name: "複数の URL が含まれる場合は全てリンク化される",
			text: "https://example.com and https://example.org",
			wantContains: []string{
				`<a href="https://example.com"`,
				`<a href="https://example.org"`,
			},
		},
		{
			name: "改行が保持される",
			text: "Line 1\nLine 2",
			wantContains: []string{
				"Line 1\nLine 2",
			},
		},
		{
			name: "改行を含むテキストの URL がリンク化される",
			text: "https://example.com\nhttps://example.org",
			wantContains: []string{
				`<a href="https://example.com"`,
				`<a href="https://example.org"`,
				"\n",
			},
		},
		{
			name: "URL 末尾の句読点はリンクから除外される",
			text: "See https://example.com.",
			wantContains: []string{
				`<a href="https://example.com"`,
				`>https://example.com</a>`,
				`.`,
			},
			wantExcludes: []string{
				`>https://example.com.</a>`,
			},
		},
		{
			name: "クエリパラメータに & を含む URL がリンク化される",
			text: "https://example.com/?a=b&c=d",
			wantContains: []string{
				`<a href="https://example.com/?a=b&amp;c=d"`,
				`>https://example.com/?a=b&amp;c=d</a>`,
			},
		},
		{
			name: "javascript: スキームはリンク化されない",
			text: "javascript:alert('xss')",
			wantContains: []string{
				"javascript:alert",
			},
			wantExcludes: []string{
				`<a href="javascript:`,
			},
		},
		{
			name:         "空文字列の場合は空文字列を返す",
			text:         "",
			wantContains: []string{},
		},
		{
			name: "URL の中の HTML 特殊文字がエスケープされる",
			text: `https://example.com/?q=<script>`,
			wantContains: []string{
				`<a href="https://example.com/?q=`,
			},
			wantExcludes: []string{
				`<a href="https://example.com/?q=<script>`,
			},
		},
		{
			name: "アングルブラケットで囲まれた URL がリンク化される",
			text: "<https://example.com>",
			wantContains: []string{
				`&lt;<a href="https://example.com" rel="noopener noreferrer" target="_blank">https://example.com</a>&gt;`,
			},
			wantExcludes: []string{
				`<a href="https://example.com&gt;`,
			},
		},
		{
			name: "丸括弧で囲まれた URL は閉じ括弧がリンクから除外される",
			text: "(https://example.com)",
			wantContains: []string{
				`(<a href="https://example.com" rel="noopener noreferrer" target="_blank">https://example.com</a>)`,
			},
			wantExcludes: []string{
				`>https://example.com)</a>`,
			},
		},
		{
			name: "Wikipedia 形式の URL(対応する括弧を含む)はリンク全体に保持される",
			text: "https://en.wikipedia.org/wiki/Foo_(bar)",
			wantContains: []string{
				`<a href="https://en.wikipedia.org/wiki/Foo_(bar)" rel="noopener noreferrer" target="_blank">https://en.wikipedia.org/wiki/Foo_(bar)</a>`,
			},
		},
		{
			name: "丸括弧で囲まれた Wikipedia 形式の URL は外側の閉じ括弧のみ除外される",
			text: "See (https://en.wikipedia.org/wiki/Foo_(bar)) for details",
			wantContains: []string{
				`See (<a href="https://en.wikipedia.org/wiki/Foo_(bar)" rel="noopener noreferrer" target="_blank">https://en.wikipedia.org/wiki/Foo_(bar)</a>) for details`,
			},
		},
		{
			name: "末尾の閉じ括弧と句読点が連続する場合は両方除外される",
			text: "See https://example.com).",
			wantContains: []string{
				`<a href="https://example.com" rel="noopener noreferrer" target="_blank">https://example.com</a>).`,
			},
			wantExcludes: []string{
				`>https://example.com).</a>`,
				`>https://example.com)</a>`,
			},
		},
		{
			name: "角括弧で囲まれた URL は閉じ括弧がリンクから除外される",
			text: "[https://example.com]",
			wantContains: []string{
				`[<a href="https://example.com" rel="noopener noreferrer" target="_blank">https://example.com</a>]`,
			},
			wantExcludes: []string{
				`>https://example.com]</a>`,
			},
		},
		{
			name: "波括弧で囲まれた URL は閉じ括弧がリンクから除外される",
			text: "{https://example.com}",
			wantContains: []string{
				`{<a href="https://example.com" rel="noopener noreferrer" target="_blank">https://example.com</a>}`,
			},
			wantExcludes: []string{
				`>https://example.com}</a>`,
			},
		},
		{
			name: "マルチバイト文字（日本語）を含む URL がリンク化される",
			text: "https://ja.wikipedia.org/wiki/日本語",
			wantContains: []string{
				`<a href="https://ja.wikipedia.org/wiki/日本語" rel="noopener noreferrer" target="_blank">https://ja.wikipedia.org/wiki/日本語</a>`,
			},
		},
		{
			name: "マルチバイト文字を含む URL の後ろに句読点がある場合",
			text: "See https://ja.wikipedia.org/wiki/日本語.",
			wantContains: []string{
				`<a href="https://ja.wikipedia.org/wiki/日本語" rel="noopener noreferrer" target="_blank">https://ja.wikipedia.org/wiki/日本語</a>.`,
			},
			wantExcludes: []string{
				`>https://ja.wikipedia.org/wiki/日本語.</a>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := LinkifyPlainText(tt.text)

			// 空文字列を期待するケース
			if tt.text == "" {
				if got != "" {
					t.Errorf("LinkifyPlainText(\"\") = %q, want \"\"", got)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("出力に %q が含まれていない\n出力: %s", want, got)
				}
			}

			for _, exclude := range tt.wantExcludes {
				if strings.Contains(got, exclude) {
					t.Errorf("出力に %q が含まれるべきではない\n出力: %s", exclude, got)
				}
			}
		})
	}
}
