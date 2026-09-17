package markup

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
)

// headingIDTransformerは、本文中の見出しにRails版と同じ規則で作ったidを付けるAST変換。
//
// Rails版はcommonmarker (comrak) のheader_ids拡張で見出しにidを付けており、本文には
// その規則を前提に `[見出しへジャンプ](#見出し)` のようなページ内リンクが書かれている。
// goldmarkの自動採番はASCII英数字を含まない見出しを `heading` の連番にしてしまい、
// それらのリンクが遷移しなくなるため、comrakの規則で付け直す。
//
// goldmarkのid生成器 (parser.IDs) を差し替えないのは、生成器に渡されるのが記法を含んだ
// 見出しの行そのもので、comrakのように記法を除いたテキストからidを作れないためである。
type headingIDTransformer struct{}

// TransformはASTを文書順に辿り、見出しにidを付ける。idの重複は1つの本文の中で避ける。
func (headingIDTransformer) Transform(document *ast.Document, reader gmtext.Reader, _ parser.Context) {
	source := reader.Source()
	anchorizer := newHeadingAnchorizer()

	_ = ast.Walk(document, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		id := anchorizer.anchorize(headingText(heading, source))
		// comrakは空のidもそのまま出力するが、何も指せない属性は付けない。
		// 空のidも使ったものとして数えるため、後に続く見出しの連番はcomrakと変わらない。
		if id != "" {
			heading.SetAttributeString("id", []byte(id))
		}

		return ast.WalkSkipChildren, nil
	})
}

// headingAnchorizerは、見出しのテキストからidを作り、1つの本文の中で重ならないようにする。
// comrak 0.41の `Anchorizer` と同じ規則である。
type headingAnchorizer struct {
	used map[string]bool
}

func newHeadingAnchorizer() *headingAnchorizer {
	return &headingAnchorizer{used: map[string]bool{}}
}

// anchorizeは、テキストを小文字にし、文字・結合文字・数字・連結用句読点・半角スペース・
// `-` だけを残して半角スペースを `-` にする。すでに使ったidと重なる場合は、使われていない
// 最初の `-1` / `-2` / … を末尾に付ける。
func (a *headingAnchorizer) anchorize(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsNumber(r) || unicode.Is(unicode.Pc, r):
			b.WriteRune(r)
		}
	}

	base := b.String()
	id := base
	for uniq := 1; a.used[id]; uniq++ {
		id = base + "-" + strconv.Itoa(uniq)
	}
	a.used[id] = true

	return id
}

// headingTextは、comrakの `collect_text` と同じように見出しのテキストを取り出す。
// テキストとインラインコードの中身をつなげ、改行は半角スペース1つとして数え、インラインの
// HTMLタグは数えない。
func headingText(heading *ast.Heading, source []byte) string {
	var escaped bytes.Buffer
	w := bufio.NewWriter(&escaped)
	collectHeadingText(heading, source, w)
	_ = w.Flush()

	// goldmarkのWriterはバックスラッシュエスケープと文字参照を解決したうえでHTMLとして
	// エスケープするため、それを戻して描画される文字にする。
	return html.UnescapeString(escaped.String())
}

// collectHeadingTextは、ノード配下のテキストを描画と同じWriterでwへ書き出す。
func collectHeadingText(n ast.Node, source []byte, w *bufio.Writer) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch node := c.(type) {
		case *ast.Text:
			if node.IsRaw() {
				gmhtml.DefaultWriter.RawWrite(w, node.Segment.Value(source))
			} else {
				gmhtml.DefaultWriter.Write(w, node.Segment.Value(source))
			}
			if node.SoftLineBreak() || node.HardLineBreak() {
				_ = w.WriteByte(' ')
			}
		case *ast.String:
			if node.IsCode() || node.IsRaw() {
				gmhtml.DefaultWriter.RawWrite(w, node.Value)
			} else {
				gmhtml.DefaultWriter.Write(w, node.Value)
			}
		case *ast.CodeSpan:
			// インラインコードの中身はエスケープを解決しない。描画と同じく改行は半角スペースにする。
			for t := node.FirstChild(); t != nil; t = t.NextSibling() {
				text, ok := t.(*ast.Text)
				if !ok {
					continue
				}
				value := bytes.TrimSuffix(text.Segment.Value(source), []byte("\n"))
				gmhtml.DefaultWriter.RawWrite(w, value)
				if len(value) != len(text.Segment.Value(source)) {
					_ = w.WriteByte(' ')
				}
			}
		case *ast.AutoLink:
			gmhtml.DefaultWriter.Write(w, node.Label(source))
		case *ast.RawHTML:
			// インラインのHTMLタグは数えない。
		default:
			collectHeadingText(c, source, w)
		}
	}
}
