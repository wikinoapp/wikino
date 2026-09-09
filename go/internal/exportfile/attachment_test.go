package exportfile

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestRewriteAttachmentLinks_ParsedLabelSyntax(t *testing.T) {
	t.Parallel()

	names := map[model.AttachmentID]string{"01ABC": "file.png"}
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
			want := "[" + tt.label + "](attachments/file.png)"
			if got := RewriteAttachmentLinks(body, names); got != want {
				t.Errorf("RewriteAttachmentLinks() = %q, want %q", got, want)
			}
		})
	}
}

func TestRewriteAttachmentLinks_DestinationSyntax(t *testing.T) {
	t.Parallel()

	names := map[model.AttachmentID]string{
		"01ABC":     "file.png",
		"01DEF":     "image.png",
		"01SPECIAL": "A=1&times \"'<`>.png",
	}
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "引用内で改行したリンク先",
			body: "> [file](\n>   /attachments/01ABC)",
			want: "> [file](\n>   attachments/file.png)",
		},
		{
			name: "入れ子の引用とリスト内で改行した画像",
			body: "> > - ![image](\n> >   </attachments/01ABC>\n> >   \"title\")",
			want: "> > - ![image](\n> >   <attachments/file.png>\n> >   \"title\")",
		},
		{
			name: "引用内で改行した参照定義",
			body: "> [file][ref]\n>\n> [ref]:\n>   /attachments/01ABC",
			want: "> [file][ref]\n>\n> [ref]:\n>   attachments/file.png",
		},
		{
			name: "リスト内で改行した画像の参照定義",
			body: "- ![image][ref]\n\n  [ref]:\n    </attachments/01ABC> \"title\"",
			want: "- ![image][ref]\n\n  [ref]:\n    <attachments/file.png> \"title\"",
		},
		{
			name: "子画像と外側リンクの両方を書き換える",
			body: `[![image](/attachments/01DEF "]")](/attachments/01ABC)`,
			want: `[![image](attachments/image.png "]")](attachments/file.png)`,
		},
		{
			name: "引用符なしの img と a 属性を維持",
			body: `<img alt=image src=/attachments/01ABC width=10> <a href=/attachments/01DEF>file</a>`,
			want: `<img alt=image src=attachments/file.png width=10> <a href=attachments/image.png>file</a>`,
		},
		{
			name: "後続属性にあるタグの例は書き換えない",
			body: `<img src=/attachments/01ABC title='<img src=/attachments/01DEF>'>`,
			want: `<img src=attachments/file.png title='<img src=/attachments/01DEF>'>`,
		},
		{
			name: "引用符なしの属性でもファイル名を安全にエスケープ",
			body: `<img src=/attachments/01SPECIAL> <a href=/attachments/01SPECIAL>file</a>`,
			want: `<img src=attachments/A%3D1&amp;times%20%22%27%3C%60%3E.png> <a href=attachments/A%3D1&amp;times%20%22%27%3C%60%3E.png>file</a>`,
		},
		{
			name: "引用符なしでも data 属性とコメントと Markdown のコードは維持",
			body: "<img data-src=/attachments/01ABC> <!-- <a href=/attachments/01ABC>file</a> -->\n\n`<img src=/attachments/01ABC>`",
			want: "<img data-src=/attachments/01ABC> <!-- <a href=/attachments/01ABC>file</a> -->\n\n`<img src=/attachments/01ABC>`",
		},
		{
			name: "raw HTML の code 要素の中の参照は書き換える",
			body: "<code><img src=/attachments/01ABC></code>",
			want: "<code><img src=attachments/file.png></code>",
		},
		{
			name: "Markdown のパーセントエンコードを復号",
			body: `[file](/attachments/%30%31ABC)`,
			want: `[file](attachments/file.png)`,
		},
		{
			name: "Markdown の文字参照とエスケープを解釈",
			body: `![image](/attachments/&#48;\%31ABC)`,
			want: `![image](attachments/file.png)`,
		},
		{
			name: "改行した参照定義のエンコードされた ID",
			body: "> [file][ref]\n>\n> [ref]:\n>   </attachments/&#x30;%31ABC>",
			want: "> [file][ref]\n>\n> [ref]:\n>   <attachments/file.png>",
		},
		{
			name: "HTML 属性の文字参照を解釈",
			body: `<img src="/attachments/&#48;1ABC"> <a href='/attachments/&#x30;%31ABC'>file</a>`,
			want: `<img src="attachments/file.png"> <a href='attachments/file.png'>file</a>`,
		},
		{
			name: "引用符なしの HTML 属性の名前付き文字参照を解釈",
			body: `<img src=/attachments/&percnt;30%31ABC>`,
			want: `<img src=attachments/file.png>`,
		},
		{
			name: "二重エンコードを別の ID として維持",
			body: `[file](/attachments/%2530%2531ABC) <img src="/attachments/&amp;#48;1ABC">`,
			want: `[file](/attachments/%2530%2531ABC) <img src="/attachments/&amp;#48;1ABC">`,
		},
		{
			name: "不正なエンコードとパス区切りを含む ID は維持",
			body: `[file](/attachments/%3ZABC) ![image](/attachments/01%2FABC) <img src=/attachments/01%5CABC>`,
			want: `[file](/attachments/%3ZABC) ![image](/attachments/01%2FABC) <img src=/attachments/01%5CABC>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := RewriteAttachmentLinks(tt.body, names); got != tt.want {
				t.Errorf("RewriteAttachmentLinks() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRewriteAttachmentLinks(t *testing.T) {
	t.Parallel()

	names := map[model.AttachmentID]string{
		"01JQ0000000000000000000001": "diagram.png",
		"01JQ0000000000000000000002": "screen shot (1).png",
		"01JQ0000000000000000000003": "図 1.png",
		"01JQ0000000000000000000004": "note#1?.png",
		"01JQ0000000000000000000005": "A&times.png",
	}

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "Markdown の画像のリンク先を相対パスへ書き換える",
			body: "![図](/attachments/01JQ0000000000000000000001)",
			want: "![図](attachments/diagram.png)",
		},
		{
			name: "Markdown のリンクのリンク先を相対パスへ書き換える",
			body: "[資料](/attachments/01JQ0000000000000000000001)",
			want: "[資料](attachments/diagram.png)",
		},
		{
			name: "img 要素の src を相対パスへ書き換える",
			body: `<img width="600" height="400" alt="図" src="/attachments/01JQ0000000000000000000001">`,
			want: `<img width="600" height="400" alt="図" src="attachments/diagram.png">`,
		},
		{
			name: "a 要素の href を相対パスへ書き換える",
			body: `<a href="/attachments/01JQ0000000000000000000001">資料</a>`,
			want: `<a href="attachments/diagram.png">資料</a>`,
		},
		{
			name: "シングルクォートの属性も書き換える",
			body: `<img src='/attachments/01JQ0000000000000000000001'>`,
			want: `<img src='attachments/diagram.png'>`,
		},
		{
			name: "空白と括弧を含むファイル名をパーセントエンコードする",
			body: "![図](/attachments/01JQ0000000000000000000002)",
			want: "![図](attachments/screen%20shot%20%281%29.png)",
		},
		{
			name: "日本語のファイル名をパーセントエンコードする",
			body: "![図](/attachments/01JQ0000000000000000000003)",
			want: "![図](attachments/%E5%9B%B3%201.png)",
		},
		{
			name: "リンクを壊す記号をパーセントエンコードする",
			body: "![図](/attachments/01JQ0000000000000000000004)",
			want: "![図](attachments/note%231%3F.png)",
		},
		{
			name: "タイトル付きのリンクではリンク先だけを書き換える",
			body: `![図](/attachments/01JQ0000000000000000000001 "図の説明")`,
			want: `![図](attachments/diagram.png "図の説明")`,
		},
		{
			name: "1 つの本文の複数の参照をまとめて書き換える",
			body: "![図](/attachments/01JQ0000000000000000000001)\n\n[資料](/attachments/01JQ0000000000000000000002)",
			want: "![図](attachments/diagram.png)\n\n[資料](attachments/screen%20shot%20%281%29.png)",
		},
		{
			name: "同じ添付ファイルへの複数の参照をすべて書き換える",
			body: "![図](/attachments/01JQ0000000000000000000001) と ![図](/attachments/01JQ0000000000000000000001)",
			want: "![図](attachments/diagram.png) と ![図](attachments/diagram.png)",
		},
		{
			name: "対応表に無い添付ファイルへの参照は原文のまま残す",
			body: "![図](/attachments/01JQ0000000000000000000009) と ![図](/attachments/01JQ0000000000000000000001)",
			want: "![図](/attachments/01JQ0000000000000000000009) と ![図](attachments/diagram.png)",
		},
		{
			name: "HTML 属性のファイル名の & を文字参照から守る",
			body: `<img src="/attachments/01JQ0000000000000000000005">`,
			want: `<img src="attachments/A&amp;times.png">`,
		},
		{
			name: "Markdown のリンク先では & をそのまま残す",
			body: "![図](/attachments/01JQ0000000000000000000005)",
			want: "![図](attachments/A&times.png)",
		},
		{
			name: "ラベルにエスケープした角括弧を含むリンクのリンク先を書き換える",
			body: `[報告書\]2026.pdf](/attachments/01JQ0000000000000000000001)`,
			want: `[報告書\]2026.pdf](attachments/diagram.png)`,
		},
		{
			name: "入れ子の画像リンクの内側と外側をどちらも書き換える",
			body: "[![図](/attachments/01JQ0000000000000000000001)](/attachments/01JQ0000000000000000000002)",
			want: "[![図](attachments/diagram.png)](attachments/screen%20shot%20%281%29.png)",
		},
		{
			name: "山括弧で囲んだリンク先を書き換える",
			body: "[資料](</attachments/01JQ0000000000000000000001>)",
			want: "[資料](<attachments/diagram.png>)",
		},
		{
			name: "フェンス付きコードブロックの中の参照は書き換えない",
			body: "```\n![図](/attachments/01JQ0000000000000000000001)\n```\n\n![図](/attachments/01JQ0000000000000000000001)",
			want: "```\n![図](/attachments/01JQ0000000000000000000001)\n```\n\n![図](attachments/diagram.png)",
		},
		{
			name: "フェンス付きコードブロックの情報文字列にある参照は書き換えない",
			body: "```![図](/attachments/01JQ0000000000000000000001)\n本文\n```\n\n![図](/attachments/01JQ0000000000000000000001)",
			want: "```![図](/attachments/01JQ0000000000000000000001)\n本文\n```\n\n![図](attachments/diagram.png)",
		},
		{
			name: "インラインコードの中の参照は書き換えない",
			body: "`![図](/attachments/01JQ0000000000000000000001)` と ![図](/attachments/01JQ0000000000000000000001)",
			want: "`![図](/attachments/01JQ0000000000000000000001)` と ![図](attachments/diagram.png)",
		},
		{
			name: "raw HTML の pre 要素の中の参照は書き換えない",
			body: "<pre>\n![図](/attachments/01JQ0000000000000000000001)\n</pre>\n\n![図](/attachments/01JQ0000000000000000000001)",
			want: "<pre>\n![図](/attachments/01JQ0000000000000000000001)\n</pre>\n\n![図](attachments/diagram.png)",
		},
		{
			name: "自己終了記法の code 要素の中と後ろにある参照を書き換える",
			body: "<code/>![図](/attachments/01JQ0000000000000000000001)</code> と ![図](/attachments/01JQ0000000000000000000001)",
			want: "<code/>![図](attachments/diagram.png)</code> と ![図](attachments/diagram.png)",
		},
		{
			name: "段落中の code 要素の中にある参照を書き換える",
			body: "本文 <code>[図](/attachments/01JQ0000000000000000000001)</code> 本文",
			want: "本文 <code>[図](attachments/diagram.png)</code> 本文",
		},
		{
			name: "pre 要素が開く HTML ブロックの中の参照は書き換えない",
			body: "<pre/>![図](/attachments/01JQ0000000000000000000001)</pre>\n\n![図](/attachments/01JQ0000000000000000000001)",
			want: "<pre/>![図](/attachments/01JQ0000000000000000000001)</pre>\n\n![図](attachments/diagram.png)",
		},
		{
			name: "段落中の pre 要素の中にある参照を書き換える",
			body: "本文 <pre/>![図](/attachments/01JQ0000000000000000000001)</pre> 本文",
			want: "本文 <pre/>![図](attachments/diagram.png)</pre> 本文",
		},
		{
			name: "HTML として解釈されない img 風の文字列は書き換えない",
			body: `\<img src="/attachments/01JQ0000000000000000000001">` + "\n\n" +
				`<!-- <img src="/attachments/01JQ0000000000000000000001"> -->` + "\n\n" +
				`<script>const image = '<img src="/attachments/01JQ0000000000000000000001">'</script>` + "\n\n" +
				`<style>.x { content: '<img src="/attachments/01JQ0000000000000000000001">' }</style>` + "\n\n" +
				`<img src="/attachments/01JQ0000000000000000000001">`,
			want: `\<img src="/attachments/01JQ0000000000000000000001">` + "\n\n" +
				`<!-- <img src="/attachments/01JQ0000000000000000000001"> -->` + "\n\n" +
				`<script>const image = '<img src="/attachments/01JQ0000000000000000000001">'</script>` + "\n\n" +
				`<style>.x { content: '<img src="/attachments/01JQ0000000000000000000001">' }</style>` + "\n\n" +
				`<img src="attachments/diagram.png">`,
		},
		{
			name: "閉じ角括弧の無いタグは後続のタグと 1 つに読まれても書き換えない",
			body: `<img src="/attachments/01JQ0000000000000000000001" alt="図"` + "\nキャプション <br>\n\n" +
				`<img src="/attachments/01JQ0000000000000000000001"> <br>`,
			want: `<img src="/attachments/01JQ0000000000000000000001" alt="図"` + "\nキャプション <br>\n\n" +
				`<img src="attachments/diagram.png"> <br>`,
		},
		{
			name: "ラベルの中で始まる破棄要素の後ろにあるリンク先を書き換える",
			body: "[<title>x](/attachments/01JQ0000000000000000000001)",
			want: "[<title>x](attachments/diagram.png)",
		},
		{
			name: "破棄要素の中で始まるリンクのリンク先は書き換えない",
			body: "<title>[x](/attachments/01JQ0000000000000000000001)</title>",
			want: "<title>[x](/attachments/01JQ0000000000000000000001)</title>",
		},
		{
			name: "段落中の raw text 要素の中の参照は書き換えない",
			body: "本文 <textarea>![図](/attachments/01JQ0000000000000000000001)</textarea> 本文\n\n" +
				"本文 <xmp>![図](/attachments/01JQ0000000000000000000001)</xmp> 本文\n\n" +
				"![図](/attachments/01JQ0000000000000000000001)",
			want: "本文 <textarea>![図](/attachments/01JQ0000000000000000000001)</textarea> 本文\n\n" +
				"本文 <xmp>![図](/attachments/01JQ0000000000000000000001)</xmp> 本文\n\n" +
				"![図](attachments/diagram.png)",
		},
		{
			name: "通常の文章にある src と Markdown の断片は書き換えない",
			body: `example src="/attachments/01JQ0000000000000000000001" and ](/attachments/01JQ0000000000000000000001) then ![図](/attachments/01JQ0000000000000000000001)`,
			want: `example src="/attachments/01JQ0000000000000000000001" and ](/attachments/01JQ0000000000000000000001) then ![図](attachments/diagram.png)`,
		},
		{
			name: "data-src 属性は書き換えない",
			body: `<div data-src="/attachments/01JQ0000000000000000000001"></div>` + "\n\n![図](/attachments/01JQ0000000000000000000001)",
			want: `<div data-src="/attachments/01JQ0000000000000000000001"></div>` + "\n\n![図](attachments/diagram.png)",
		},
		{
			name: "参照リンクのリンク先を定義側で書き換える",
			body: "[資料][ref]\n\n[ref]: /attachments/01JQ0000000000000000000001",
			want: "[資料][ref]\n\n[ref]: attachments/diagram.png",
		},
		{
			name: "参照画像のリンク先を定義側で書き換える",
			body: "![図][ref]\n\n[ref]: /attachments/01JQ0000000000000000000001",
			want: "![図][ref]\n\n[ref]: attachments/diagram.png",
		},
		{
			name: "タイトル付きのリンク参照定義ではリンク先だけを書き換える",
			body: `[資料][ref]` + "\n\n" + `[ref]: /attachments/01JQ0000000000000000000003 "説明"`,
			want: `[資料][ref]` + "\n\n" + `[ref]: attachments/%E5%9B%B3%201.png "説明"`,
		},
		{
			// 表示側の ExtractAttachmentIDs も拾わないためアーカイブに複製が入らない
			name: "サニタイズが中身ごと落とす要素の中の参照は書き換えない",
			body: `<object><img src="/attachments/01JQ0000000000000000000001"></object>` + "\n\n" +
				`![図](/attachments/01JQ0000000000000000000001)`,
			want: `<object><img src="/attachments/01JQ0000000000000000000001"></object>` + "\n\n" +
				`![図](attachments/diagram.png)`,
		},
		{
			name: "破棄要素を異なる破棄要素の終了タグで閉じた後の参照は書き換える",
			body: `<object><img src="/attachments/01JQ0000000000000000000001"></iframe>` + "\n\n" +
				`![図](/attachments/01JQ0000000000000000000001)`,
			want: `<object><img src="/attachments/01JQ0000000000000000000001"></iframe>` + "\n\n" +
				`![図](attachments/diagram.png)`,
		},
		{
			name: "破棄要素の中の script 要素の終了タグより後ろは書き換えない",
			body: `<object><img src="/attachments/01JQ0000000000000000000001"></script>` + "\n\n" +
				`![図](/attachments/01JQ0000000000000000000001)`,
			want: `<object><img src="/attachments/01JQ0000000000000000000001"></script>` + "\n\n" +
				`![図](/attachments/01JQ0000000000000000000001)`,
		},
		{
			name: "入れ子の a 要素の href も相対パスへ書き換える",
			body: `<a href="/attachments/01JQ0000000000000000000001"><a href="/attachments/01JQ0000000000000000000001"></a>`,
			want: `<a href="attachments/diagram.png"><a href="attachments/diagram.png"></a>`,
		},
		{
			name: "ラベルのコードスパンに角括弧を含むリンクのリンク先を書き換える",
			body: "[コード `]` の図](/attachments/01JQ0000000000000000000001)",
			want: "[コード `]` の図](attachments/diagram.png)",
		},
		{
			name: "対象外要素の src と href 属性は書き換えない",
			body: `<video src="/attachments/01JQ0000000000000000000001"></video><link href="/attachments/01JQ0000000000000000000001">` + "\n\n![図](/attachments/01JQ0000000000000000000001)",
			want: `<video src="/attachments/01JQ0000000000000000000001"></video><link href="/attachments/01JQ0000000000000000000001">` + "\n\n![図](attachments/diagram.png)",
		},
		{
			// img タグだけの行は HTML ブロックを開かないため、続く行も画面と同じく Markdown として
			// 読まれる
			name: "img タグの行に続く Markdown の参照も書き換える",
			body: `<img src="/attachments/01JQ0000000000000000000001">` + "\n*キャプション*\n" +
				"![図](/attachments/01JQ0000000000000000000001)",
			want: `<img src="attachments/diagram.png">` + "\n*キャプション*\n" +
				"![図](attachments/diagram.png)",
		},
		{
			name: "img タグの行に続くコードの中の参照は書き換えない",
			body: `<img src="/attachments/01JQ0000000000000000000001">` + "\n*キャプション*\n" +
				"`![図](/attachments/01JQ0000000000000000000001)`",
			want: `<img src="attachments/diagram.png">` + "\n*キャプション*\n" +
				"`![図](/attachments/01JQ0000000000000000000001)`",
		},
		{
			name: "パスの区切りを文字参照で書いた参照も書き換える",
			body: "[a](&#47;attachments/01JQ0000000000000000000001)",
			want: "[a](attachments/diagram.png)",
		},
		{
			name: "パスの区切りをエスケープで書いた参照も書き換える",
			body: `[a](\/attachments/01JQ0000000000000000000001)`,
			want: "[a](attachments/diagram.png)",
		},
		{
			name: "HTML 属性のパスの区切りを文字参照で書いた参照も書き換える",
			body: `<img src="&#47;attachments/01JQ0000000000000000000001">`,
			want: `<img src="attachments/diagram.png">`,
		},
		{
			name: "添付ファイルを参照しない本文はそのまま返す",
			body: "[外部リンク](https://example.com/attachments) と本文。",
			want: "[外部リンク](https://example.com/attachments) と本文。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RewriteAttachmentLinks(tt.body, names)
			if got != tt.want {
				t.Errorf("RewriteAttachmentLinks(%q, names) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

func TestRewriteAttachmentLinksWithoutNames(t *testing.T) {
	t.Parallel()

	body := "![図](/attachments/01JQ0000000000000000000001)"

	got := RewriteAttachmentLinks(body, nil)
	if got != body {
		t.Errorf("RewriteAttachmentLinks(%q, nil) = %q, want %q", body, got, body)
	}
}

func TestRewriteAttachmentLinks_RenderedElements(t *testing.T) {
	t.Parallel()

	names := map[model.AttachmentID]string{"01A": "a.png", "01B": "b.png"}
	tests := []struct {
		name string
		body string
		want string
		ids  []model.AttachmentID
	}{
		{
			name: "image after an incomplete tag",
			body: "<img src=\"/attachments/01A\"\n<img src=\"/attachments/01B\">",
			want: "<img src=\"/attachments/01A\"\n<img src=\"attachments/b.png\">",
			ids:  []model.AttachmentID{"01B"},
		},
		{
			name: "anchor after an incomplete tag",
			body: "<img src=\"/attachments/01A\"\n<a href=\"/attachments/01B\">file</a>",
			want: "<img src=\"/attachments/01A\"\n<a href=\"attachments/b.png\">file</a>",
			ids:  []model.AttachmentID{"01B"},
		},
		{
			name: "image after an unclosed attribute quote",
			body: "<img title=\"\n<img src='/attachments/01B'>",
			want: "<img title=\"\n<img src='attachments/b.png'>",
			ids:  []model.AttachmentID{"01B"},
		},
		{
			name: "recovered raw text still hides its content",
			body: "<img\n<textarea><img src=/attachments/01A></textarea> <img src=/attachments/01B>",
			want: "<img\n<textarea><img src=/attachments/01A></textarea> <img src=attachments/b.png>",
			ids:  []model.AttachmentID{"01B"},
		},
		{
			name: "image label raw text cannot hide the next image",
			body: "![<textarea>x](/attachments/01A) ![next](/attachments/01B)",
			want: "![<textarea>x](attachments/a.png) ![next](attachments/b.png)",
			ids:  []model.AttachmentID{"01A", "01B"},
		},
		{
			name: "image label self-closing raw text cannot hide the next image",
			body: "![<textarea/>x](/attachments/01A) <img src=/attachments/01B>",
			want: "![<textarea/>x](attachments/a.png) <img src=attachments/b.png>",
			ids:  []model.AttachmentID{"01A", "01B"},
		},
		{
			name: "image label discarded element cannot hide the next image",
			body: "![<object>x](/attachments/01A) ![next](/attachments/01B)",
			want: "![<object>x](attachments/a.png) ![next](attachments/b.png)",
			ids:  []model.AttachmentID{"01A", "01B"},
		},
		{
			name: "image label raw image is not a reference",
			body: "![<img src=\"/attachments/01A\">](/attachments/01B)",
			want: "![<img src=\"/attachments/01A\">](attachments/b.png)",
			ids:  []model.AttachmentID{"01B"},
		},
		{
			name: "image label Markdown link is not a reference",
			body: "![alt [link](/attachments/01A)](/attachments/01B)",
			want: "![alt [link](/attachments/01A)](attachments/b.png)",
			ids:  []model.AttachmentID{"01B"},
		},
		{
			name: "image label nested image is not a reference",
			body: "![alt ![inner](/attachments/01A)](/attachments/01B)",
			want: "![alt ![inner](/attachments/01A)](attachments/b.png)",
			ids:  []model.AttachmentID{"01B"},
		},
		{
			name: "external image destination still excludes its label children",
			body: "![alt [link](/attachments/01A)](https://example.com/image.png)",
			want: "![alt [link](/attachments/01A)](https://example.com/image.png)",
		},
		{
			name: "reference image excludes its label children",
			body: "![alt [link](/attachments/01A)][image]\n\n[image]: /attachments/01B",
			want: "![alt [link](/attachments/01A)][image]\n\n[image]: attachments/b.png",
			ids:  []model.AttachmentID{"01B"},
		},
		{
			name: "image inside a real link preserves both destinations",
			body: "[![alt <object>](/attachments/01A)](/attachments/01B)",
			want: "[![alt <object>](attachments/a.png)](attachments/b.png)",
			ids:  []model.AttachmentID{"01A", "01B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matches := markup.ScanAttachmentRefMatches(tt.body)
			if len(matches) != len(tt.ids) {
				t.Fatalf("ScanAttachmentRefMatches() = %v, want IDs %v", matches, tt.ids)
			}
			for i, match := range matches {
				if match.AttachmentID != tt.ids[i] {
					t.Errorf("match[%d].AttachmentID = %q, want %q", i, match.AttachmentID, tt.ids[i])
				}
				if got := tt.body[match.Start:match.Stop]; got != "/attachments/"+string(tt.ids[i]) {
					t.Errorf("match[%d] source = %q, want /attachments/%s", i, got, tt.ids[i])
				}
			}
			if got := RewriteAttachmentLinks(tt.body, names); got != tt.want {
				t.Errorf("RewriteAttachmentLinks() = %q, want %q", got, tt.want)
			}
		})
	}
}
