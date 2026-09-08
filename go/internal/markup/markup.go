// Package markup はMarkdownテキストをサニタイズ済みHTMLに変換する機能を提供する。
package markup

import (
	"log/slog"
	"reflect"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	gmtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// imgBlockLineRegex matches a line whose whole content is a single img start tag.
//
// [Ja] imgBlockLineRegex は、行の内容が img 開始タグ 1 つだけである行に一致する。
var imgBlockLineRegex = regexp.MustCompile(`(?i)^[ ]{0,3}<img\b(?:[^>"']|"[^"]*"|'[^']*')*>[ \t]*(?:\r\n|\n)?$`)

// inlineImgBlockParser is goldmark's HTML block parser with one line shape taken away from it: a
// line holding nothing but an img tag is left to the paragraph parser instead of opening a block.
//
// An img is an inline element, but a block it opens ends only at a blank line. A caption written on
// the next line would be swallowed by that block and reach the reader as unparsed source, so a
// "*caption*" under an img tag would show its asterisks. Reading the tag as inline HTML instead
// also gives a raw img the shape the image syntax of Markdown renders to.
//
// The line itself decides this, rather than a rewrite of the body before parsing, so that every
// caller reading a body through this parser sees the structure the screen shows.
//
// [Ja] inlineImgBlockParser は、1 つの行の形だけを取り除いた goldmark の HTML ブロックパーサー。
// img タグだけの行は HTML ブロックを開かず、段落パーサーに残す。
//
// img はインライン要素だが、これが開くブロックは空行までしか終わらない。次の行に書いた
// キャプションはそのブロックに飲み込まれ、解析されないソースのまま読み手に届くため、img タグの
// 下の "*caption*" はアスタリスクごと表示されてしまう。タグをインライン HTML として読めば、
// raw な img も Markdown の画像記法がレンダリングされる形と同じになる。
//
// 解析前に本文を書き換えるのではなく行そのもので判断するのは、このパーサーを通して本文を読む
// 呼び出し元がどれも画面の見せる構造を見られるようにするためである。
type inlineImgBlockParser struct {
	parser.BlockParser
}

// Open leaves a line holding only an img tag unclaimed, so that the parsers after it read the line.
//
// [Ja] Open は img タグだけの行を引き受けない。そうすることで後続のパーサーがその行を読む。
func (p *inlineImgBlockParser) Open(
	parent ast.Node,
	reader gmtext.Reader,
	pc parser.Context,
) (ast.Node, parser.State) {
	if line, _ := reader.PeekLine(); imgBlockLineRegex.Match(line) {
		return nil, parser.NoChildren
	}

	return p.BlockParser.Open(parent, reader, pc)
}

// withInlineImgBlocks is the parser option that puts inlineImgBlockParser where goldmark registers
// its own HTML block parser, keeping the priority that place carries.
//
// [Ja] withInlineImgBlocks は、goldmark が自身の HTML ブロックパーサーを登録している場所へ
// inlineImgBlockParser を置くパーサーオプション。その場所が持つ優先度はそのまま引き継ぐ。
type withInlineImgBlocks struct{}

// SetParserOption implements parser.Option. The parser to wrap is found by type rather than by
// identity: goldmark hands out one shared instance of it today, and one built per call would still
// have the same type.
//
// Finding none of them panics, which happens while this package builds its parser and therefore
// before anything renders. The alternative is a body parsed by an option that quietly did nothing,
// where every line holding only an img tag opens a block again.
//
// [Ja] SetParserOption は parser.Option を実装する。包む対象のパーサーは同一性ではなく型で探す。
// goldmark は現在そのインスタンスを 1 つ共有して返すが、呼び出しごとに作るようになっても型は
// 変わらないためである。
//
// 見つからない場合は panic する。これは本パッケージがパーサーを組み立てる時点、つまり何かを
// レンダリングするより前に起きる。そうしなければ、黙って何もしなかったオプションのまま本文が
// 解析され、img タグだけの行が再びブロックを開くことになる。
func (o withInlineImgBlocks) SetParserOption(c *parser.Config) {
	htmlBlockParserType := reflect.TypeOf(parser.NewHTMLBlockParser())
	for i, value := range c.BlockParsers {
		blockParser, ok := value.Value.(parser.BlockParser)
		if !ok || reflect.TypeOf(value.Value) != htmlBlockParserType {
			continue
		}

		c.BlockParsers[i] = util.Prioritized(
			&inlineImgBlockParser{BlockParser: blockParser},
			value.Priority,
		)

		return
	}

	panic("goldmarkのHTMLブロックパーサーが登録されていないため、imgタグだけの行の扱いを差し替えられません")
}

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.Linkify,
		extension.NewTable(
			extension.WithTableCellAlignMethod(extension.TableCellAlignAttribute),
		),
		extension.Strikethrough,
		extension.TaskList,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		withInlineImgBlocks{},
	),
	goldmark.WithRendererOptions(
		// Rails版の parse: {html: true}, render: {unsafe: true} に相当。
		// Markdown内のHTMLタグをそのまま出力する。
		html.WithUnsafe(),
		html.WithHardWraps(),
	),
)

// policy はHTMLサニタイズポリシー。
// bluemondayのUGCPolicy（User Generated Content向け）をベースに、
// タスクリスト用のinput要素とimg要素のwidth/height属性を許可する。
var policy = newSanitizationPolicy()

func newSanitizationPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	// タスクリスト記法（- [ ] / - [x]）で生成されるチェックボックスを許可
	p.AllowAttrs("type", "checked", "disabled").OnElements("input")

	// img要素のwidth/height属性を許可（Rails版のsanitization_configに相当）
	p.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")

	// テーブルの配置指定（GFM）で生成されるalign属性を許可
	p.AllowAttrs("align").OnElements("td", "th")

	return p
}

// contentDiscardedElements is the set of HTML elements the sanitization policy drops together with
// everything they enclose, counting how many of them it has entered as it goes. It is the default
// set bluemonday discards the content of when the element itself is not allowed, less the elements
// of rawTextDiscardedElements, which it reads before reaching that counter. This policy allows none
// of them, so nothing written inside one reaches the screen.
//
// A caller reading a body the way the screen shows it has to leave these out: what they enclose is
// neither text nor code of the page, it is content that is never rendered at all. Since that
// counter is what ends the discarded run, an end tag named after any element of this set closes one
// level of it, even when another element of the set opened that level.
//
// [Ja] contentDiscardedElements は、サニタイズポリシーが囲まれたものごと落とす HTML 要素のセット。
// ポリシーはいくつ入ったかを数えながら落とす。bluemonday が、要素自体を許可していないときに中身も
// 捨てる既定の集合から、そのカウンターより手前で読まれる rawTextDiscardedElements の要素を除いた
// ものである。本ポリシーはこれらを 1 つも許可していないため、中に書かれたものは画面に届かない。
//
// 画面の見え方どおりに本文を読む側は、これらを対象から外す必要がある。囲まれているものはページの
// テキストでもコードでもなく、そもそもレンダリングされない内容だからである。捨てられている範囲を
// 終わらせるのはこのカウンターであるため、この集合のいずれかの名前を持つ終了タグは 1 段階を閉じる。
// その段階を開いたのが集合の別の要素であっても変わらない。
var contentDiscardedElements = map[string]bool{
	"frame":    true,
	"frameset": true,
	"iframe":   true,
	"noembed":  true,
	"noframes": true,
	"noscript": true,
	"nostyle":  true,
	"object":   true,
	"title":    true,
}

// rawTextDiscardedElements is the set of HTML elements whose content the sanitization policy drops
// without passing them through the counter of contentDiscardedElements. bluemonday reads a script
// or a style tag before that counter and leaves it untouched, dropping instead the raw text the
// element holds.
//
// An end tag of one therefore closes only its own element, and never a run of content discarded by
// an element of contentDiscardedElements. A caller leaves what these enclose out for the same
// reason it leaves out the elements above, since the screen shows none of it.
//
// [Ja] rawTextDiscardedElements は、サニタイズポリシーが中身を落とすものの、
// contentDiscardedElements のカウンターを通さない HTML 要素のセット。bluemonday は script タグ・
// style タグをそのカウンターより手前で読んでカウンターに触れず、代わりに要素が持つ raw text を
// 落とす。
//
// そのため、これらの終了タグが閉じるのは自身の要素だけであり、contentDiscardedElements の要素が
// 捨てている範囲を閉じることはない。呼び出し元がこれらの囲む範囲を対象から外す理由は上の要素と
// 同じで、画面にはそのどれも出ないためである。
var rawTextDiscardedElements = map[string]bool{
	"script": true,
	"style":  true,
}

// rawTextEscapedElements is the set of HTML elements whose content the HTML parser reads as raw
// text and the sanitization policy then leaves behind as ordinary text. Markdown written after one
// of their start tags is still parsed, so a link there becomes an a element while the body is
// rendered, but that element ends up as text of the raw-text element and reaches the reader as the
// notation itself rather than as a link.
//
// A caller reading a body the way the screen shows it therefore treats what follows such a start
// tag as text rather than as a reference to something. The raw-text elements the policy handles
// through its counters are left out: script and style are rawTextDiscardedElements, and title,
// iframe, noembed, noframes and noscript are contentDiscardedElements.
//
// [Ja] rawTextEscapedElements は、HTML パーサーが中身を raw text として読み、サニタイズポリシーが
// それを通常のテキストとして残す HTML 要素のセット。開始タグより後ろの Markdown は解析されるため、
// そこにあるリンクはレンダリングの過程で a 要素になるが、その要素は raw text 要素のテキストとして
// 収まり、リンクではなく記法そのものとして読み手に届く。
//
// 画面の見え方どおりに本文を読む側は、そうした開始タグの後ろを何かへの参照ではなくテキストとして
// 扱う。ポリシーがカウンターで処理する raw text 要素は含めない。script と style は
// rawTextDiscardedElements に、title・iframe・noembed・noframes・noscript は
// contentDiscardedElements にある。
var rawTextEscapedElements = map[string]bool{
	"plaintext": true,
	"textarea":  true,
	"xmp":       true,
}

// RenderMarkdown はMarkdownテキストをサニタイズ済みHTMLに変換する。
// 空文字列が渡された場合は空文字列を返す。
func RenderMarkdown(body string) string {
	_, _, bodyHTML := renderBody(body)

	return bodyHTML
}

// renderBody renders a body the way RenderMarkdown does and hands back the parse alongside the
// HTML, so that a caller needing to read the body again does not parse it a second time.
//
// The source it returns is the body after the newline normalization, which is what the parse and
// the offsets of every node refer to.
//
// [Ja] renderBody は RenderMarkdown と同じように本文をレンダリングし、HTML とあわせて解析結果を
// 返す。本文をもう一度読む必要のある呼び出し元が、2 度目の解析を行わずに済むようにするためである。
//
// 返すソースは改行を正規化した後の本文で、解析結果と各ノードの位置が指しているのはこちらである。
func renderBody(body string) ([]byte, ast.Node, string) {
	if body == "" {
		return nil, nil, ""
	}

	// CRLFをLFに正規化（ブラウザのtextareaはCRLFで送信する場合がある）
	body = strings.ReplaceAll(body, "\r\n", "\n")

	source := []byte(body)
	document := md.Parser().Parse(gmtext.NewReader(source))
	bodyHTML, err := renderSanitized(source, document)
	if err != nil {
		slog.Warn("Markdown変換に失敗", "error", err)

		return nil, nil, ""
	}

	return source, document, bodyHTML
}
