// Package markupはMarkdownテキストをサニタイズ済みHTMLに変換する機能を提供する。
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

// imgBlockLineRegexは、行の内容がimg開始タグ1つだけである行に一致する。
var imgBlockLineRegex = regexp.MustCompile(`(?i)^[ ]{0,3}<img\b(?:[^>"']|"[^"]*"|'[^']*')*>[ \t]*(?:\r\n|\n)?$`)

// inlineImgBlockParserは、1つの行の形だけを取り除いたgoldmarkのHTMLブロックパーサー。
// imgタグだけの行はHTMLブロックを開かず、段落パーサーに残す。
//
// imgはインライン要素だが、これが開くブロックは空行までしか終わらない。次の行に書いた
// キャプションはそのブロックに飲み込まれ、解析されないソースのまま読み手に届くため、imgタグの
// 下の "*caption*" はアスタリスクごと表示されてしまう。タグをインラインHTMLとして読めば、
// rawなimgもMarkdownの画像記法がレンダリングされる形と同じになる。
//
// 解析前に本文を書き換えるのではなく行そのもので判断するのは、このパーサーを通して本文を読む
// 呼び出し元がどれも画面の見せる構造を見られるようにするためである。
type inlineImgBlockParser struct {
	parser.BlockParser
}

// Openはimgタグだけの行を引き受けない。そうすることで後続のパーサーがその行を読む。
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

// withInlineImgBlocksは、goldmarkが自身のHTMLブロックパーサーを登録している場所へ
// inlineImgBlockParserを置くパーサーオプション。その場所が持つ優先度はそのまま引き継ぐ。
type withInlineImgBlocks struct{}

// SetParserOptionはparser.Optionを実装する。包む対象のパーサーは同一性ではなく型で探す。
// goldmarkは現在そのインスタンスを1つ共有して返すが、呼び出しごとに作るようになっても型は
// 変わらないためである。
//
// 見つからない場合はpanicする。これは本パッケージがパーサーを組み立てる時点、つまり何かを
// レンダリングするより前に起きる。そうしなければ、黙って何もしなかったオプションのまま本文が
// 解析され、imgタグだけの行が再びブロックを開くことになる。
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
		// Rails版のparse: {html: true}, render: {unsafe: true} に相当。
		// Markdown内のHTMLタグをそのまま出力する。
		html.WithUnsafe(),
		html.WithHardWraps(),
	),
)

// policyはHTMLサニタイズポリシー。
// bluemondayのUGCPolicy (User Generated Content向け) をベースに、
// タスクリスト用のinput要素とimg要素のwidth/height属性を許可する。
var policy = newSanitizationPolicy()

func newSanitizationPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	// タスクリスト記法 (- [ ] / - [x]) で生成されるチェックボックスを許可
	p.AllowAttrs("type", "checked", "disabled").OnElements("input")

	// img要素のwidth/height属性を許可 (Rails版のsanitization_configに相当)
	p.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")

	// テーブルの配置指定 (GFM) で生成されるalign属性を許可
	p.AllowAttrs("align").OnElements("td", "th")

	return p
}

// contentDiscardedElementsは、サニタイズポリシーが囲まれたものごと落とすHTML要素のセット。
// ポリシーはいくつ入ったかを数えながら落とす。bluemondayが、要素自体を許可していないときに中身も
// 捨てる既定の集合から、そのカウンターより手前で読まれるrawTextDiscardedElementsの要素を除いた
// ものである。本ポリシーはこれらを1つも許可していないため、中に書かれたものは画面に届かない。
//
// 画面の見え方どおりに本文を読む側は、これらを対象から外す必要がある。囲まれているものはページの
// テキストでもコードでもなく、そもそもレンダリングされない内容だからである。捨てられている範囲を
// 終わらせるのはこのカウンターであるため、この集合のいずれかの名前を持つ終了タグは1段階を閉じる。
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

// rawTextDiscardedElementsは、サニタイズポリシーが中身を落とすものの、
// contentDiscardedElementsのカウンターを通さないHTML要素のセット。bluemondayはscriptタグ・
// styleタグをそのカウンターより手前で読んでカウンターに触れず、代わりに要素が持つraw textを
// 落とす。
//
// そのため、これらの終了タグが閉じるのは自身の要素だけであり、contentDiscardedElementsの要素が
// 捨てている範囲を閉じることはない。呼び出し元がこれらの囲む範囲を対象から外す理由は上の要素と
// 同じで、画面にはそのどれも出ないためである。
var rawTextDiscardedElements = map[string]bool{
	"script": true,
	"style":  true,
}

// rawTextEscapedElementsは、HTMLパーサーが中身をraw textとして読み、サニタイズポリシーが
// それを通常のテキストとして残すHTML要素のセット。開始タグより後ろのMarkdownは解析されるため、
// そこにあるリンクはレンダリングの過程でa要素になるが、その要素はraw text要素のテキストとして
// 収まり、リンクではなく記法そのものとして読み手に届く。
//
// 画面の見え方どおりに本文を読む側は、そうした開始タグの後ろを何かへの参照ではなくテキストとして
// 扱う。ポリシーがカウンターで処理するraw text要素は含めない。scriptとstyleは
// rawTextDiscardedElementsに、title・iframe・noembed・noframes・noscriptは
// contentDiscardedElementsにある。
var rawTextEscapedElements = map[string]bool{
	"plaintext": true,
	"textarea":  true,
	"xmp":       true,
}

// RenderMarkdownはMarkdownテキストをサニタイズ済みHTMLに変換する。
// 空文字列が渡された場合は空文字列を返す。
func RenderMarkdown(body string) string {
	_, _, bodyHTML := renderBody(body)

	return bodyHTML
}

// renderBodyはRenderMarkdownと同じように本文をレンダリングし、HTMLとあわせて解析結果を
// 返す。本文をもう一度読む必要のある呼び出し元が、2度目の解析を行わずに済むようにするためである。
//
// 返すソースは改行を正規化した後の本文で、解析結果と各ノードの位置が指しているのはこちらである。
func renderBody(body string) ([]byte, ast.Node, string) {
	if body == "" {
		return nil, nil, ""
	}

	// CRLFをLFに正規化 (ブラウザのtextareaはCRLFで送信する場合がある)
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
