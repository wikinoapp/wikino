package markup

import (
	"bytes"
	"crypto/rand"
	"log/slog"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
)

// wikilinkSourceMarkers connects source text to its position in the rendered HTML tree. Markers
// are plain text added after Markdown parsing, alongside text containing literal [[ and inside
// every piece of Markdown code. They neither introduce elements nor turn whitespace-only table
// content into foster-parented text. The augmented document is used only for this scan and never
// returned to a caller.
//
// [Ja] wikilinkSourceMarkers はソースのテキストと、描画後の HTML ツリー内の位置を対応づける。
// マーカーは Markdown 解析後に、リテラルの [[ を含むテキストと Markdown のコードのそれぞれへ
// 加える通常の文字列である。要素を追加せず、空白だけの表の内容が表の外へ移動する原因にもならない。
// マーカーを加えた文書はこの走査にだけ使い、呼び出し元へは返さない。
type wikilinkSourceMarkers struct {
	prefix string
	ranges [][]byteRange

	// alwaysProtected holds source no marker can stand for, because the renderer writes none of
	// it anywhere: whatever follows the language in the info string of a fence.
	//
	// [Ja] alwaysProtected は、どのマーカーでも表せないソースを持つ。レンダラーがそれをどこにも
	// 書かないためで、フェンスの情報文字列で言語の後ろに続く部分がこれに当たる
	alwaysProtected []byteRange
}

// add records the source ranges one marker stands for and returns the marker. A random prefix
// keeps author-written text, including text decoded from character references, from being mistaken
// for an internal marker.
//
// A marker can stand for several ranges because a piece of Markdown syntax that reaches the reader
// as one thing occupies several of them: the lines of a code block and the info string of its
// fence share the fate of the element the block renders to.
//
// [Ja] add はマーカー 1 つが表すソース範囲を記録してマーカーを返す。ランダムな接頭辞により、
// 文字参照から復号したテキストも含め、書き手のテキストを内部マーカーと取り違えることを防ぐ。
//
// 1 つのマーカーが複数の範囲を表せるのは、読み手に 1 つのものとして届く Markdown の構文が複数の
// 範囲を占めるためである。コードブロックの各行とフェンスの情報文字列は、そのブロックが
// レンダリングされる要素と運命を共にする。
func (m *wikilinkSourceMarkers) add(sourceRanges ...byteRange) string {
	index := len(m.ranges)
	m.ranges = append(m.ranges, sourceRanges)

	return m.prefix + strconv.Itoa(index) + "Z"
}

// scanRenderedWikilinkRanges uses the same renderer, sanitizer and fragment parser as display to
// locate text inside skipped elements. In particular, HTML tree construction decides where raw
// code/pre/a elements end when Markdown-generated containers and tables intervene, and a raw-text
// element decides whether the code Markdown wrote reaches the reader as a code element at all.
// Source syntax and source spelling are still handled by scanWikilinkSourceRanges, separately from
// this mapping. document is local to the scan; marking it does not change the stored Markdown.
//
// [Ja] scanRenderedWikilinkRanges は表示と同じレンダラー・サニタイザー・フラグメントパーサーを使い、
// 変換を省略する要素内のテキストを特定する。Markdown が生成したコンテナーや表が挟まる場合も、
// raw な code/pre/a 要素の終わりは HTML ツリー構築で決まり、Markdown が書いたコードが code 要素
// として読み手に届くかどうかは raw text 要素が決める。ソースの構文と表記の判定は、この対応づけ
// とは別に scanWikilinkSourceRanges が引き続き行う。
// document は走査専用であり、マーカーを加えても保存済みの Markdown は変わらない。
func scanRenderedWikilinkRanges(source []byte, document ast.Node) []byteRange {
	markers := wikilinkSourceMarkers{prefix: "Wikino" + rand.Text()}
	augmentedSource := source
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Image:
			return ast.WalkSkipChildren, nil
		case *ast.CodeSpan:
			augmentedSource = markers.markCodeSpan(augmentedSource, n)

			return ast.WalkSkipChildren, nil
		case *ast.FencedCodeBlock:
			augmentedSource = markers.markCodeBlock(augmentedSource, source, n, n.Info)

			return ast.WalkSkipChildren, nil
		case *ast.CodeBlock:
			augmentedSource = markers.markCodeBlock(augmentedSource, source, n, nil)

			return ast.WalkSkipChildren, nil
		case *ast.Text:
			// Markdown can split the opening brackets across adjacent Text nodes.
			// Include one following byte when finding the node that starts the pair.
			//
			// [Ja] Markdown は開き角括弧を隣り合う Text ノードに分けることがある。
			// その組を開始するノードを見つけるときは、直後の 1 バイトも含める。
			stop := min(n.Segment.Stop+1, len(source))
			if bytes.Contains(source[n.Segment.Start:stop], []byte(wikilinkOpening)) {
				marker := markers.add(byteRange{start: n.Segment.Start, stop: n.Segment.Stop})
				n.Parent().InsertBefore(n.Parent(), n, ast.NewString([]byte(marker)))
			}
		case *ast.HTMLBlock:
			augmentedSource = markers.markHTMLBlock(augmentedSource, source, n)
		}

		return ast.WalkContinue, nil
	})
	if len(markers.ranges) == 0 {
		return markers.alwaysProtected
	}

	container, err := renderedTree(augmentedSource, document)
	if err != nil {
		slog.Warn("Failed to read wiki-link source markers", "error", err)

		return []byteRange{{start: 0, stop: len(source)}}
	}

	visible := make([]bool, len(markers.ranges))
	markers.findVisible(container, false, visible)
	protected := markers.alwaysProtected
	for i, sourceRanges := range markers.ranges {
		if !visible[i] {
			protected = append(protected, sourceRanges...)
		}
	}

	return protected
}

// markCodeSpan marks a code span. The marker goes inside the span rather than beside it, since a
// raw-text element opened earlier turns the whole element into text the reader can read.
//
// The marker becomes a text child of its own, because the renderer reads the children of a code
// span as ast.Text nodes and leaves no place for a node of another kind.
//
// Every span is marked, not only the ones holding brackets. The ranges of a span also split the
// text around it, and the screen splits it at exactly the same place: where a code element stands.
//
// [Ja] markCodeSpan はコードスパンにマーカーを付ける。マーカーを隣ではなく中に置くのは、手前で
// 開かれた raw text 要素が要素ごと読めるテキストへ変えてしまうためである。
//
// マーカーは自身のテキストの子になる。レンダラーがコードスパンの子を ast.Text として読むため、
// 別種のノードを差し込む場所が無いからである。
//
// 角括弧を持つスパンだけでなくすべてのスパンにマーカーを付ける。スパンの範囲はその前後の
// テキストを分割する役割も持ち、画面が分割するのも同じ場所、すなわち code 要素のある位置である。
func (m *wikilinkSourceMarkers) markCodeSpan(augmented []byte, node *ast.CodeSpan) []byte {
	var ranges []byteRange
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			ranges = append(ranges, byteRange{start: textNode.Segment.Start, stop: textNode.Segment.Stop})
		}
	}
	if len(ranges) == 0 {
		return augmented
	}

	augmented, segment := appendMarker(augmented, m.add(ranges...))
	node.InsertBefore(node, node.FirstChild(), ast.NewTextSegment(segment))

	return augmented
}

// markCodeBlock marks a code block, together with the info string of a fence when it has one. The
// two share one marker: the info string reaches the reader only in the same case the lines do,
// which is when the block is not rendered as a code element after all.
//
// The marker becomes a line of its own where the block has lines, so that the source of the block
// is not copied. A fence with an info string but no lines carries it in the info string instead.
//
// [Ja] markCodeBlock はコードブロックに、フェンスであれば情報文字列とあわせてマーカーを付ける。
// 両者でマーカーを 1 つ共有する。情報文字列が読み手に届くのは各行が届くのと同じ場合、つまり
// ブロックが結局 code 要素としてレンダリングされない場合だけだからである。
//
// 行を持つブロックではマーカーを 1 行として加え、ブロックのソースを複製しない。行を持たず
// 情報文字列だけを持つフェンスでは、代わりに情報文字列へ付ける。
func (m *wikilinkSourceMarkers) markCodeBlock(
	augmented []byte,
	source []byte,
	node ast.Node,
	info *ast.Text,
) []byte {
	var ranges []byteRange
	if info != nil {
		language, rest := fenceInfoRanges(source, info.Segment)
		if language.start < language.stop {
			ranges = append(ranges, language)
		}
		if rest.start < rest.stop {
			m.alwaysProtected = append(m.alwaysProtected, rest)
		}
	}
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		ranges = append(ranges, byteRange{start: segment.Start, stop: segment.Stop})
	}
	if len(ranges) == 0 {
		return augmented
	}

	marker := m.add(ranges...)
	if lines.Len() > 0 {
		augmented, segment := appendMarker(augmented, marker)
		lines.Unshift(segment)

		return augmented
	}

	start := len(augmented)
	augmented = append(augmented, marker...)
	augmented = append(augmented, info.Segment.Value(source)...)
	info.Segment = gmtext.NewSegment(start, len(augmented))

	return augmented
}

// fenceInfoRanges splits the info string of a fence into the language, which the renderer writes
// into the class attribute of the code element, and the rest, which it writes nowhere. Only the
// language can reach the reader, and then only where the element itself becomes text.
//
// The split is the renderer's: the language ends at the first space. A segment carrying padding
// starts with the spaces the renderer writes in front of it, so its language is empty.
//
// [Ja] fenceInfoRanges はフェンスの情報文字列を、レンダラーが code 要素の class 属性へ書く言語
// と、どこにも書かない残りとに分ける。読み手に届きうるのは言語だけで、それも要素自体がテキスト
// になる場合に限られる。
//
// 分け方はレンダラーと同じで、言語は最初の空白で終わる。パディングを持つセグメントは、レンダラー
// がその前に書く空白から始まるため、言語は空になる。
func fenceInfoRanges(source []byte, segment gmtext.Segment) (byteRange, byteRange) {
	info := byteRange{start: segment.Start, stop: segment.Stop}
	if segment.Padding > 0 {
		return byteRange{start: info.start, stop: info.start}, info
	}

	stop := info.stop
	if space := bytes.IndexByte(source[info.start:info.stop], ' '); space >= 0 {
		stop = info.start + space
	}

	return byteRange{start: info.start, stop: stop}, byteRange{start: stop, stop: info.stop}
}

// markableHTMLToken reports whether a marker can be written inside an HTML token without changing
// what the token is. Text, a comment and a doctype carry a payload the marker joins, while a tag
// carries attributes a marker would become one of.
//
// A comment has to be markable because a raw-text element opened earlier turns it into text the
// reader can read, and the brackets written inside it then reach the screen like any other text.
// Nothing else tells that apart: the scan of the raw HTML tokens stops at the raw-text element,
// which is where the comment stops being a comment.
//
// [Ja] markableHTMLToken は、HTML トークンの中へマーカーを書いてもそのトークンが何であるかが
// 変わらないかを返す。テキスト・コメント・doctype はマーカーが加わる中身を持つが、タグが持つのは
// 属性であり、マーカーはその 1 つになってしまう。
//
// コメントを対象にする必要があるのは、手前で開かれた raw text 要素がコメントを読めるテキストへ
// 変えてしまい、その中に書かれた角括弧がほかのテキストと同じように画面へ届くためである。これを
// 見分けられるものはほかにない。raw HTML のトークンの走査は raw text 要素で止まるが、そこは
// まさにコメントがコメントでなくなる位置である。
func markableHTMLToken(tokenType html.TokenType) bool {
	switch tokenType {
	case html.TextToken, html.CommentToken, html.DoctypeToken:
		return true
	default:
		return false
	}
}

// appendMarker appends marker to augmented and returns the segment naming what it wrote.
//
// [Ja] appendMarker は marker を augmented の末尾へ書き、書いた範囲を指すセグメントを返す。
func appendMarker(augmented []byte, marker string) ([]byte, gmtext.Segment) {
	start := len(augmented)
	augmented = append(augmented, marker...)

	return augmented, gmtext.NewSegment(start, len(augmented))
}

// markHTMLBlock inserts markers in the tokens that carry a payload rather than markup, retaining
// the mapping across source lines whose blockquote or list prefixes the Markdown renderer omits.
// The original segments are joined before tokenization so multiline tags and raw-text elements
// retain their normal interpretation.
//
// [Ja] markHTMLBlock はマークアップではなく中身を運ぶトークンにマーカーを挿入する。Markdown の
// レンダラーが引用・リストの接頭辞を除く行でもソースとの対応を保つ。トークナイズ前に元の
// セグメントを連結することで、複数行のタグや raw text 要素を通常どおり解釈できる。
func (m *wikilinkSourceMarkers) markHTMLBlock(augmented, source []byte, node *ast.HTMLBlock) []byte {
	type sourcePiece struct {
		rendered byteRange
		start    int
	}
	var raw bytes.Buffer
	var pieces []sourcePiece
	appendSegment := func(segment gmtext.Segment) {
		start := raw.Len() + segment.Padding
		pieces = append(pieces, sourcePiece{
			rendered: byteRange{start: start, stop: start + segment.Stop - segment.Start},
			start:    segment.Start,
		})
		raw.Write(segment.Value(source))
	}
	for i := 0; i < node.Lines().Len(); i++ {
		appendSegment(node.Lines().At(i))
	}
	if node.HasClosure() {
		appendSegment(node.ClosureLine)
	}
	if !bytes.Contains(raw.Bytes(), []byte(wikilinkOpening)) {
		return augmented
	}

	var marked bytes.Buffer
	tokenizer := html.NewTokenizer(bytes.NewReader(raw.Bytes()))
	position, written, pieceIndex := 0, 0, 0
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}
		start := position
		position += len(tokenizer.Raw())
		if !markableHTMLToken(tokenType) {
			continue
		}
		for pieceIndex < len(pieces) && pieces[pieceIndex].rendered.stop <= start {
			pieceIndex++
		}
		for i := pieceIndex; i < len(pieces) && pieces[i].rendered.start < position; i++ {
			piece := pieces[i]
			from, to := max(start, piece.rendered.start), min(position, piece.rendered.stop)
			opening := bytes.Index(raw.Bytes()[from:to], []byte(wikilinkOpening))
			if opening < 0 {
				continue
			}
			marked.Write(raw.Bytes()[written : from+opening])
			marked.WriteString(m.add(byteRange{
				start: piece.start + from - piece.rendered.start,
				stop:  piece.start + to - piece.rendered.start,
			}))
			written = from + opening
		}
	}
	if marked.Len() == 0 {
		return augmented
	}
	marked.Write(raw.Bytes()[written:])
	start := len(augmented)
	augmented = append(augmented, marked.Bytes()...)
	node.Lines().Clear()
	node.Lines().Append(gmtext.NewSegment(start, len(augmented)))
	node.ClosureLine = gmtext.NewSegment(-1, -1)

	return augmented
}

// findVisible records markers in ordinary text nodes. Missing markers belong to content the
// sanitizer dropped, while markers under skipped elements belong to text display never converts.
//
// [Ja] findVisible は通常のテキストノードにあるマーカーを記録する。消えたマーカーはサニタイザーが
// 破棄した内容に属し、除外要素の下のマーカーは表示側が変換しないテキストに属する。
func (m *wikilinkSourceMarkers) findVisible(node *html.Node, inSkip bool, visible []bool) {
	inSkip = inSkip || (node.Type == html.ElementNode && skipElements[node.Data])
	if node.Type == html.TextNode && !inSkip {
		remaining := node.Data
		for {
			_, after, found := strings.Cut(remaining, m.prefix)
			if !found {
				break
			}
			digits, rest, found := strings.Cut(after, "Z")
			if !found {
				break
			}
			if index, err := strconv.Atoi(digits); err == nil && index >= 0 && index < len(visible) {
				visible[index] = true
			}
			remaining = rest
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		m.findVisible(child, inSkip, visible)
	}
}
