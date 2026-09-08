package exportfile

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/markup"
)

func TestRewriteWikilinks_ParsedLabelSyntax(t *testing.T) {
	t.Parallel()

	paths := map[PageRef]string{
		{TopicName: "T", PageTitle: "A"}: "T/A.md",
		{TopicName: "T", PageTitle: "B"}: "T/B.md",
	}
	tests := []struct {
		name  string
		label string
	}{
		{name: "HTML 属性の閉じ角括弧", label: `<span title="]">label</span> [[A]]`},
		{name: "HTML 属性の開き角括弧", label: `<span title="[">label</span> [[A]]`},
		{name: "HTML コメントの閉じ角括弧", label: `<!-- ] -->label [[A]]`},
		{name: "HTML コメントの開き角括弧", label: `<!-- [ -->label [[A]]`},
		{name: "子画像のリンク先の閉じ角括弧", label: `![image](https://example.com/image]x.png) [[A]]`},
		{name: "子画像のリンク先の開き角括弧", label: `![image](https://example.com/image[x.png) [[A]]`},
		{name: "子画像のタイトルの閉じ角括弧", label: `![image](https://example.com/image.png "]") [[A]]`},
		{name: "子画像のタイトルの開き角括弧", label: `![image](https://example.com/image.png "[") [[A]]`},
		{name: "子画像の中の HTML", label: `![<span title="]">image</span>](https://example.com/image.png) [[A]]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			link := "[" + tt.label + "](https://example.com/)"
			got := RewriteWikilinks(link+" [[B]]", "T", paths)
			want := link + " [[T/B.md]]"
			if got != want {
				t.Errorf("RewriteWikilinks() = %q, want %q", got, want)
			}
		})
	}
}

func TestRewriteWikilinks(t *testing.T) {
	t.Parallel()

	paths := map[PageRef]string{
		{TopicName: "設計", PageTitle: "API"}:    "設計/API.md",
		{TopicName: "設計", PageTitle: "API|一覧"}: "設計/API｜一覧-2.md",
		{TopicName: "運用", PageTitle: "デプロイ手順"}: "運用/デプロイ手順.md",
		{TopicName: "運用", PageTitle: "C#"}:     "運用/C＃.md",
	}

	tests := []struct {
		name             string
		body             string
		currentTopicName string
		want             string
	}{
		{
			name:             "トピック名付きのリンクを対応表のパスへ書き換える",
			body:             "詳細は [[運用/デプロイ手順]] を参照。",
			currentTopicName: "設計",
			want:             "詳細は [[運用/デプロイ手順.md]] を参照。",
		},
		{
			name:             "省略形のリンクに本文のトピック名を補う",
			body:             "詳細は [[API]] を参照。",
			currentTopicName: "設計",
			want:             "詳細は [[設計/API.md]] を参照。",
		},
		{
			name:             "全角へ置き換えられ連番の付いたファイル名を引く",
			body:             "[[設計/API|一覧]] と [[運用/C#]]",
			currentTopicName: "設計",
			want:             "[[設計/API｜一覧-2.md]] と [[運用/C＃.md]]",
		},
		{
			name:             "対応表に無いページへのリンクは原文のまま残す",
			body:             "[[存在しないページ]] と [[設計/API]]",
			currentTopicName: "設計",
			want:             "[[存在しないページ]] と [[設計/API.md]]",
		},
		{
			name:             "省略形が別のトピックの同名ページを指すことはない",
			body:             "[[デプロイ手順]]",
			currentTopicName: "設計",
			want:             "[[デプロイ手順]]",
		},
		{
			name:             "フェンス付きコードブロックの中のリンクは書き換えない",
			body:             "```\n[[設計/API]]\n```\n\n[[設計/API]]",
			currentTopicName: "設計",
			want:             "```\n[[設計/API]]\n```\n\n[[設計/API.md]]",
		},
		{
			name:             "フェンス付きコードブロックの情報文字列にあるリンクは書き換えない",
			body:             "```[[設計/API]]\n本文\n```\n\n[[設計/API]]",
			currentTopicName: "設計",
			want:             "```[[設計/API]]\n本文\n```\n\n[[設計/API.md]]",
		},
		{
			name:             "インラインコードの中のリンクは書き換えない",
			body:             "`[[設計/API]]` は [[設計/API]] と書く。",
			currentTopicName: "設計",
			want:             "`[[設計/API]]` は [[設計/API.md]] と書く。",
		},
		{
			name:             "字下げされたコードブロックの中のリンクは書き換えない",
			body:             "段落。\n\n    [[設計/API]]\n\n[[設計/API]]",
			currentTopicName: "設計",
			want:             "段落。\n\n    [[設計/API]]\n\n[[設計/API.md]]",
		},
		{
			name:             "raw HTML の code 要素の中のリンクは書き換えない",
			body:             "<code>[[設計/API]]</code> と [[設計/API]]",
			currentTopicName: "設計",
			want:             "<code>[[設計/API]]</code> と [[設計/API.md]]",
		},
		{
			name:             "raw HTML の pre 要素の中のリンクは書き換えない",
			body:             "<pre>\n[[設計/API]]\n</pre>\n\n[[設計/API]]",
			currentTopicName: "設計",
			want:             "<pre>\n[[設計/API]]\n</pre>\n\n[[設計/API.md]]",
		},
		{
			name:             "自己終了記法の code 要素の後ろにあるリンクは書き換えない",
			body:             "<code/>[[設計/API]]</code> と [[設計/API]]",
			currentTopicName: "設計",
			want:             "<code/>[[設計/API]]</code> と [[設計/API.md]]",
		},
		{
			name:             "自己終了記法の pre 要素の後ろにあるリンクは書き換えない",
			body:             "<pre/>[[設計/API]]</pre>\n\n[[設計/API]]",
			currentTopicName: "設計",
			want:             "<pre/>[[設計/API]]</pre>\n\n[[設計/API.md]]",
		},
		{
			name:             "サニタイズが中身ごと落とす要素の中のリンクは書き換えない",
			body:             "<iframe>[[運用/デプロイ手順]]</iframe>\n\n<object>[[運用/デプロイ手順]]</object>\n\n[[運用/デプロイ手順]]",
			currentTopicName: "設計",
			want:             "<iframe>[[運用/デプロイ手順]]</iframe>\n\n<object>[[運用/デプロイ手順]]</object>\n\n[[運用/デプロイ手順.md]]",
		},
		{
			name:             "破棄要素を異なる破棄要素の終了タグで閉じた後のリンクは書き換える",
			body:             "<object>[[設計/API]]</iframe>[[設計/API]]",
			currentTopicName: "設計",
			want:             "<object>[[設計/API]]</iframe>[[設計/API.md]]",
		},
		{
			name:             "破棄要素の中の script 要素の終了タグより後ろは書き換えない",
			body:             "<object>[[設計/API]]</script>[[設計/API]]",
			currentTopicName: "設計",
			want:             "<object>[[設計/API]]</script>[[設計/API]]",
		},
		{
			name:             "入れ子の a 要素の終了タグより後ろは書き換える",
			body:             `<a href="https://example.com/">[[設計/API]]<a href="https://example.com/">[[設計/API]]</a>[[設計/API]]`,
			currentTopicName: "設計",
			want:             `<a href="https://example.com/">[[設計/API]]<a href="https://example.com/">[[設計/API]]</a>[[設計/API.md]]`,
		},
		{
			name:             "raw HTML の構文は保ち通常要素の本文にあるリンクは書き換える",
			body:             `<div data-page="[[設計/API]]">[[設計/API]]</div>` + "\n\n<!-- [[設計/API]] -->\n\n[[設計/API]]",
			currentTopicName: "設計",
			want:             `<div data-page="[[設計/API]]">[[設計/API.md]]</div>` + "\n\n<!-- [[設計/API]] -->\n\n[[設計/API.md]]",
		},
		{
			name:             "Markdown リンクのリンク先にあるリンクは書き換えない",
			body:             "[外部](https://example.com/[[設計/API]]) と [[設計/API]]",
			currentTopicName: "設計",
			want:             "[外部](https://example.com/[[設計/API]]) と [[設計/API.md]]",
		},
		{
			name:             "山括弧付き Markdown リンク先の丸括弧より後ろは書き換えない",
			body:             "[外部](<https://example.com/a)/[[設計/API]]>) と [[設計/API]]",
			currentTopicName: "設計",
			want:             "[外部](<https://example.com/a)/[[設計/API]]>) と [[設計/API.md]]",
		},
		{
			name:             "山括弧付き Markdown リンク先の開き丸括弧より後ろは書き換えない",
			body:             "[外部](<https://example.com/a([[設計/API]]>) と [[設計/API]]",
			currentTopicName: "設計",
			want:             "[外部](<https://example.com/a([[設計/API]]>) と [[設計/API.md]]",
		},
		{
			name:             "Markdown リンクのラベルにあるリンクは書き換えない",
			body:             "[参照 [[設計/API]]](https://example.com/) と [[設計/API]]",
			currentTopicName: "設計",
			want:             "[参照 [[設計/API]]](https://example.com/) と [[設計/API.md]]",
		},
		{
			name:             "ラベルのコードスパンより後ろもラベルとして書き換えない",
			body:             "[コード `]` と [[設計/API]]](https://example.com/) と [[設計/API]]",
			currentTopicName: "設計",
			want:             "[コード `]` と [[設計/API]]](https://example.com/) と [[設計/API.md]]",
		},
		{
			name:             "リンク参照定義のリンク先にあるリンクは書き換えない",
			body:             "[a][ref]\n\n[ref]: /url/[[設計/API]]\n\n[[設計/API]]",
			currentTopicName: "設計",
			want:             "[a][ref]\n\n[ref]: /url/[[設計/API]]\n\n[[設計/API.md]]",
		},
		{
			name:             "ショートカット参照の直後のリンクは書き換える",
			body:             "[a][[設計/API]]\n\n[a]: /url",
			currentTopicName: "設計",
			want:             "[a][[設計/API.md]]\n\n[a]: /url",
		},
		{
			name:             "raw HTML の a 要素の中のリンクは書き換えない",
			body:             `<a href="https://example.com/">[[設計/API]]</a> と [[設計/API]]`,
			currentTopicName: "設計",
			want:             `<a href="https://example.com/">[[設計/API]]</a> と [[設計/API.md]]`,
		},
		{
			name:             "サニタイズで除去される a 要素の異なる終了タグ後は書き換える",
			body:             "<a>[[設計/API]]</code>[[設計/API]]",
			currentTopicName: "設計",
			want:             "<a>[[設計/API.md]]</code>[[設計/API.md]]",
		},
		{
			name:             "raw HTML の script 要素の中のリンクは書き換えない",
			body:             `<script>const page = "[[設計/API]]"</script>` + "\n\n[[設計/API]]",
			currentTopicName: "設計",
			want:             `<script>const page = "[[設計/API]]"</script>` + "\n\n[[設計/API.md]]",
		},
		{
			name:             "raw HTML の style 要素の中のリンクは書き換えない",
			body:             `<style>/* [[設計/API]] */</style>` + "\n\n[[設計/API]]",
			currentTopicName: "設計",
			want:             `<style>/* [[設計/API]] */</style>` + "\n\n[[設計/API.md]]",
		},
		{
			name:             "コードブロックに raw text の要素を見せていても後続の code 要素の中のリンクは書き換えない",
			body:             "```html\n<style>\n```\n\n<code>[[設計/API]]</code> と [[設計/API]]",
			currentTopicName: "設計",
			want:             "```html\n<style>\n```\n\n<code>[[設計/API]]</code> と [[設計/API.md]]",
		},
		{
			name:             "1 行に複数のリンクがあってもすべて書き換える",
			body:             "[[API]]、[[運用/デプロイ手順]]、[[API]]",
			currentTopicName: "設計",
			want:             "[[設計/API.md]]、[[運用/デプロイ手順.md]]、[[設計/API.md]]",
		},
		{
			name:             "リンクの前後の空白を保つ",
			body:             "  [[API]]  ",
			currentTopicName: "設計",
			want:             "  [[設計/API.md]]  ",
		},
		{
			name:             "リンクの中の余分な空白を取り除いて引く",
			body:             "[[ 設計/API ]]",
			currentTopicName: "設計",
			want:             "[[設計/API.md]]",
		},
		{
			name:             "中身の無いリンクはそのまま残す",
			body:             "[[]] と [[API]]",
			currentTopicName: "設計",
			want:             "[[]] と [[設計/API.md]]",
		},
		{
			// img タグだけの行は HTML ブロックを開かないため、続く行も画面と同じく Markdown として
			// 読まれる。コードスパンと既存リンクのラベルはそこでも保護される
			name: "img タグの行に続くコードと既存リンクのラベルは書き換えない",
			body: `<img src="https://example.com/i.png">` + "\n*キャプション*\n" +
				"`[[API]]` と [参照 [[API]]](https://example.com/) と [[API]]",
			currentTopicName: "設計",
			want: `<img src="https://example.com/i.png">` + "\n*キャプション*\n" +
				"`[[API]]` と [参照 [[API]]](https://example.com/) と [[設計/API.md]]",
		},
		{
			// サニタイズが受け付けないリンク先の a 要素は落ち、ラベルは通常テキストとして画面に出る
			name:             "サニタイズが落とすリンクのラベルは書き換える",
			body:             "[ラベル [[API]]](tel:+81-3-0000-0000)",
			currentTopicName: "設計",
			want:             "[ラベル [[設計/API.md]]](tel:+81-3-0000-0000)",
		},
		{
			name:             "サニタイズが落とすリンクのリンク先は書き換えない",
			body:             "[ラベル](tel:[[API]])",
			currentTopicName: "設計",
			want:             "[ラベル](tel:[[API]])",
		},
		{
			// エスケープと文字参照は角括弧そのものであって、リンクを開く構文ではない
			name:             "エスケープや文字参照で書いた角括弧のリンクは書き換えない",
			body:             `\[\[API]] と &#91;&#91;API]] と [[API]]`,
			currentTopicName: "設計",
			want:             `\[\[API]] と &#91;&#91;API]] と [[設計/API.md]]`,
		},
		{
			name:             "Wiki リンクを含まない本文はそのまま返す",
			body:             "リンクの無い本文。",
			currentTopicName: "設計",
			want:             "リンクの無い本文。",
		},
		{
			name:             "対応の取れていない開始括弧と保護範囲の後ろのリンクを書き換える",
			body:             "[[ と書く `x` [[API]] と [リンク](/u) [[運用/デプロイ手順]]",
			currentTopicName: "設計",
			want:             "[[ と書く `x` [[設計/API.md]] と [リンク](/u) [[運用/デプロイ手順.md]]",
		},
		{
			name:             "保護範囲をまたぐリンクは書き換えない",
			body:             "[[設計/`x`API]]",
			currentTopicName: "設計",
			want:             "[[設計/`x`API]]",
		},
		{
			name:             "閉じていない a 要素をリンクが閉じた後のリンクを書き換える",
			body:             `<a href="/u">x <https://example.com> [[API]] と [リンク](/v) [[運用/デプロイ手順]]`,
			currentTopicName: "設計",
			want:             `<a href="/u">x <https://example.com> [[設計/API.md]] と [リンク](/v) [[運用/デプロイ手順.md]]`,
		},
		{
			name:             "閉じていない a 要素の中のリンクは書き換えない",
			body:             `<a href="/u">x ![図](/v) [[API]]`,
			currentTopicName: "設計",
			want:             `<a href="/u">x ![図](/v) [[API]]`,
		},
		{
			name:             "引用の中の閉じていない pre 要素の外のリンクを書き換える",
			body:             "> x <pre>y\n\n> z [[API]]",
			currentTopicName: "設計",
			want:             "> x <pre>y\n\n> z [[設計/API.md]]",
		},
		{
			name:             "表のセルの中の閉じていない code 要素の外のリンクを書き換える",
			body:             "| a |\n| --- |\n| <code>y |\n\nz [[API]]",
			currentTopicName: "設計",
			want:             "| a |\n| --- |\n| <code>y |\n\nz [[設計/API.md]]",
		},
		{
			name:             "段落の中の閉じていない pre 要素の後ろのリンクは書き換えない",
			body:             "x <pre>y\n\nz [[API]]",
			currentTopicName: "設計",
			want:             "x <pre>y\n\nz [[API]]",
		},
		{
			name:             "HTML として読まれなかったタグの中のリンクを書き換える",
			body:             "before <!-- [[API]] </script> after",
			currentTopicName: "設計",
			want:             "before <!-- [[設計/API.md]] </script> after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RewriteWikilinks(tt.body, tt.currentTopicName, paths)
			if got != tt.want {
				t.Errorf("RewriteWikilinks(%q, %q, paths) = %q, want %q", tt.body, tt.currentTopicName, got, tt.want)
			}
		})
	}
}

func TestRewriteWikilinksWithoutPaths(t *testing.T) {
	t.Parallel()

	body := "[[設計/API]]"

	got := RewriteWikilinks(body, "設計", nil)
	if got != body {
		t.Errorf("RewriteWikilinks(%q, %q, nil) = %q, want %q", body, "設計", got, body)
	}
}

func TestRewriteWikilinks_MultilineDestinations(t *testing.T) {
	t.Parallel()

	paths := map[PageRef]string{
		{TopicName: "T", PageTitle: "A"}: "T/A.md",
		{TopicName: "T", PageTitle: "B"}: "T/B.md",
	}

	tests := []struct {
		name string
		link string
	}{
		{name: "引用内の閉じ丸括弧", link: "> [x](\n> <https://example.com/a)/[[A]]>)"},
		{name: "引用内の開き丸括弧", link: "> [x](\n> <https://example.com/a([[A]]>)"},
		{name: "CRLF の引用", link: "> [x](\r\n> <https://example.com/a)/[[A]]>)"},
		{name: "入れ子の引用", link: "> > [x](\n> > <https://example.com/a([[A]]>)"},
		{name: "引用内のリスト", link: "> - [x](\n>   <https://example.com/a)/[[A]]>)"},
		{name: "リストの継続行", link: "- [x](\n  <https://example.com/a([[A]]>)"},
		{name: "タイトル付きの引用", link: "> [x](\n> <https://example.com/a)/[[A]]> \"title ) [[A]]\")"},
		{name: "引用内の画像", link: "> ![x](\n> <https://example.com/a([[A]]>)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RewriteWikilinks(tt.link+" [[B]]", "T", paths)
			want := tt.link + " [[T/B.md]]"
			if got != want {
				t.Errorf("RewriteWikilinks() = %q, want %q", got, want)
			}
		})
	}
}

func TestRewriteWikilinks_DestinationEscapes(t *testing.T) {
	t.Parallel()

	paths := map[PageRef]string{
		{TopicName: "T", PageTitle: "A"}: "T/A.md",
		{TopicName: "T", PageTitle: "B"}: "T/B.md",
	}

	tests := []struct {
		name        string
		destination string
		keepsLink   bool
	}{
		{name: "通常の許可 URL", destination: "https://example.com/[[A]]", keepsLink: true},
		{name: "エスケープした許可 URL", destination: `https\://example.com/[[A]]`, keepsLink: true},
		{name: "数値文字参照の許可 URL", destination: "https&#58;//example.com/[[A]]", keepsLink: true},
		{name: "名前付き文字参照の許可 URL", destination: "https&colon;//example.com/[[A]]", keepsLink: true},
		{name: "通常の拒否 URL", destination: "tel:+81-3-0000-0000/[[A]]"},
		{name: "エスケープした拒否 URL", destination: `tel\:+81-3-0000-0000/[[A]]`},
		{name: "数値文字参照の拒否 URL", destination: "tel&#58;+81-3-0000-0000/[[A]]"},
		{name: "名前付き文字参照の拒否 URL", destination: "tel&colon;+81-3-0000-0000/[[A]]"},
	}

	for _, tt := range tests {
		for _, reference := range []bool{false, true} {
			name := tt.name
			if reference {
				name += "/参照リンク"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				label := "x [[A]]"
				if !tt.keepsLink {
					label = "x [[T/A.md]]"
				}
				body := "[x [[A]]](" + tt.destination + ") [[B]]"
				want := "[" + label + "](" + tt.destination + ") [[T/B.md]]"
				if reference {
					body = "[x [[A]]][ref] [[B]]\n\n[ref]: " + tt.destination
					want = "[" + label + "][ref] [[T/B.md]]\n\n[ref]: " + tt.destination
				}
				if got := RewriteWikilinks(body, "T", paths); got != want {
					t.Errorf("RewriteWikilinks() = %q, want %q", got, want)
				}
			})
		}
	}
}

func TestRewriteWikilinks_EscapedOpening(t *testing.T) {
	t.Parallel()

	paths := map[PageRef]string{
		{TopicName: "T", PageTitle: "A"}: "T/A.md",
		{TopicName: "T", PageTitle: "B"}: "T/B.md",
	}

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "開始括弧のエスケープ", body: "\\[[A]] と [[B]]", want: "\\[[A]] と [[T/B.md]]"},
		{name: "バックスラッシュのエスケープ", body: "\\\\[[A]] と [[B]]", want: "\\\\[[T/A.md]] と [[T/B.md]]"},
		{name: "3 本のバックスラッシュ", body: "\\\\\\[[A]] と [[B]]", want: "\\\\\\[[A]] と [[T/B.md]]"},
		{name: "両方の開始括弧のエスケープ", body: "\\[\\[A]] と [[B]]", want: "\\[\\[A]] と [[T/B.md]]"},
		{name: "文字参照の開始括弧", body: "&#91;&#91;A]] &lbrack;&lbrack;A]] [[B]]", want: "&#91;&#91;A]] &lbrack;&lbrack;A]] [[T/B.md]]"},
		{name: "HTML ブロック内のバックスラッシュ", body: "<div>\n\\[[A]]\n</div>\n\n[[B]]", want: "<div>\n\\[[T/A.md]]\n</div>\n\n[[T/B.md]]"},
		{name: "HTML ブロック内の 3 本のバックスラッシュ", body: "<div>\n\\\\\\[[A]]\n</div>\n\n[[B]]", want: "<div>\n\\\\\\[[T/A.md]]\n</div>\n\n[[T/B.md]]"},
		{name: "引用の HTML ブロック", body: "> <div>\n> \\[[A]]\n> </div>\n\n[[B]]", want: "> <div>\n> \\[[T/A.md]]\n> </div>\n\n[[T/B.md]]"},
		{name: "HTML ブロックの後の Markdown", body: "<div>\\[[A]]</div>\n\n\\[[A]] [[B]]", want: "<div>\\[[T/A.md]]</div>\n\n\\[[A]] [[T/B.md]]"},
		{name: "インライン HTML 内の Markdown", body: "本文 <span>\\[[A]]</span> [[B]]", want: "本文 <span>\\[[A]]</span> [[T/B.md]]"},
		{name: "拒否されたリンクのラベル", body: "[x \\[[A]]](tel:+81-3-0000-0000) [[B]]", want: "[x \\[[A]]](tel:+81-3-0000-0000) [[T/B.md]]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := RewriteWikilinks(tt.body, "T", paths); got != tt.want {
				t.Errorf("RewriteWikilinks() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRewriteWikilinks_RenderedElementBoundaries(t *testing.T) {
	t.Parallel()

	paths := map[PageRef]string{
		{TopicName: "T", PageTitle: "A"}: "T/A.md",
		{TopicName: "T", PageTitle: "B"}: "T/B.md",
		{TopicName: "T", PageTitle: "C"}: "T/C.md",
	}
	tests := []struct {
		name   string
		body   string
		want   string
		titles []string
	}{
		{
			name: "independent code after a quote",
			body: "> <pre>\n> [[A]]\n\n<code>[[B]]",
			want: "> <pre>\n> [[A]]\n\n<code>[[B]]",
		},
		{
			name: "independent script after a quote",
			body: "> <pre>\n> [[A]]\n\n<script>[[B]]",
			want: "> <pre>\n> [[A]]\n\n<script>[[B]]",
		},
		{
			name:   "late end tag cannot extend a quote boundary",
			body:   "> <pre>\n> [[A]]\n\n[[B]]\n\n</pre>\n\n[[C]]",
			want:   "> <pre>\n> [[A]]\n\n[[T/B.md]]\n\n</pre>\n\n[[T/C.md]]",
			titles: []string{"B", "C"},
		},
		{
			name:   "independent table cells",
			body:   "| x | y |\n| - | - |\n| <code>[[A]] | <code>[[B]] |\n\n[[C]]",
			want:   "| x | y |\n| - | - |\n| <code>[[A]] | <code>[[B]] |\n\n[[T/C.md]]",
			titles: []string{"C"},
		},
		{
			name:   "expired same-name elements before a late end tag",
			body:   "| x | y |\n| - | - |\n| <code>[[A]] | <code>[[B]] |\n\n</code> [[C]]",
			want:   "| x | y |\n| - | - |\n| <code>[[A]] | <code>[[B]] |\n\n</code> [[T/C.md]]",
			titles: []string{"C"},
		},
		{
			name:   "nested quote ends before the outer quote",
			body:   "> > <pre>\n> > [[A]]\n>\n> [[B]]\n\n[[C]]",
			want:   "> > <pre>\n> > [[A]]\n>\n> [[T/B.md]]\n\n[[T/C.md]]",
			titles: []string{"B", "C"},
		},
		{
			name:   "later pre in the outer quote gets its own boundary",
			body:   "> > <pre>\n> > [[A]]\n>\n> <pre>\n> [[B]]\n\n[[C]]",
			want:   "> > <pre>\n> > [[A]]\n>\n> <pre>\n> [[B]]\n\n[[T/C.md]]",
			titles: []string{"C"},
		},
		{
			name:   "code after an incomplete tag",
			body:   "<img\n<code>[[A]]</code> [[B]]",
			want:   "<img\n<code>[[A]]</code> [[T/B.md]]",
			titles: []string{"B"},
		},
		{
			name:   "code after an unclosed attribute quote",
			body:   "<img title=\"\n<code>[[A]]</code> [[B]]",
			want:   "<img title=\"\n<code>[[A]]</code> [[T/B.md]]",
			titles: []string{"B"},
		},
		{
			name:   "multiple incomplete tags",
			body:   "<img\n<code>[[A]]</code> <img\n<code>[[B]]</code> [[C]]",
			want:   "<img\n<code>[[A]]</code> <img\n<code>[[B]]</code> [[T/C.md]]",
			titles: []string{"C"},
		},
		{
			name:   "image label HTML cannot protect following text",
			body:   "![<code>[[A]]](/attachments/01A) [[B]]",
			want:   "![<code>[[A]]](/attachments/01A) [[T/B.md]]",
			titles: []string{"B"},
		},
		{
			name:   "image label link cannot close an outer anchor",
			body:   "<a href=\"/x\">![alt [link](/attachments/01A)](/attachments/01B) [[A]]</a> [[B]]",
			want:   "<a href=\"/x\">![alt [link](/attachments/01A)](/attachments/01B) [[A]]</a> [[T/B.md]]",
			titles: []string{"B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matches := markup.ScanWikilinkMatches(tt.body, "T")
			if len(matches) != len(tt.titles) {
				t.Fatalf("ScanWikilinkMatches() = %v, want titles %v", matches, tt.titles)
			}
			for i, match := range matches {
				if match.Key.TopicName != "T" || match.Key.PageTitle != tt.titles[i] {
					t.Errorf("match[%d].Key = %v, want T/%s", i, match.Key, tt.titles[i])
				}
				if got := tt.body[match.Start:match.Stop]; got != "[["+tt.titles[i]+"]]" {
					t.Errorf("match[%d] source = %q, want [[%s]]", i, got, tt.titles[i])
				}
			}
			if got := RewriteWikilinks(tt.body, "T", paths); got != tt.want {
				t.Errorf("RewriteWikilinks() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRewriteWikilinks_ElementScopeEndTags(t *testing.T) {
	t.Parallel()

	paths := map[PageRef]string{
		{TopicName: "T", PageTitle: "A"}: "T/A.md",
		{TopicName: "T", PageTitle: "B"}: "T/B.md",
	}
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "囲む要素の終了タグが中の pre を終わらせる",
			body: "<div><pre></div>[[A]]",
			want: "<div><pre></div>[[T/A.md]]",
		},
		{
			name: "対応しない終了タグは何も閉じない",
			body: "<h1></li><pre></h1>[[A]]",
			want: "<h1></li><pre></h1>[[T/A.md]]",
		},
		{
			name: "formatting element は開き直される",
			body: "<div><code></div>[[A]]",
			want: "<div><code></div>[[A]]",
		},
		{
			name: "表の後の終了タグは手前へ届かない",
			body: "<div></div><code><table></code>[[A]]",
			want: "<div></div><code><table></code>[[A]]",
		},
		{
			name: "規則を持たない終了タグは special 要素で止まる",
			body: "<span><pre></span>[[A]]",
			want: "<span><pre></span>[[A]]",
		},
		{
			name: "表の外のセルは何も開かない",
			body: `<code><td><img src="/x"></code>[[A]]`,
			want: `<code><td><img src="/x"></code>[[T/A.md]]`,
		},
		{
			name: "表の中のセルの終了タグ",
			body: "<table><td><code>[[A]]</code></td></table>[[B]]",
			want: "<table><td><code>[[A]]</code></td></table>[[T/B.md]]",
		},
		{
			name: "raw text の中の Markdown リンクは a を閉じない",
			body: `<a href="/x"><textarea>[l](/x)[[A]]`,
			want: `<a href="/x"><textarea>[l](/x)[[A]]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := RewriteWikilinks(tt.body, "T", paths); got != tt.want {
				t.Errorf("RewriteWikilinks() = %q, want %q", got, tt.want)
			}
		})
	}
}
