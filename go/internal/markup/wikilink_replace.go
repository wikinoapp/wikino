package markup

import (
	"crypto/rand"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// resolvedWikilinkは本文のWikiリンク1件と、それが解決するページ。
type resolvedWikilink struct {
	match    WikilinkMatch
	location *PageLocation
}

// wikilinkReplacerは本文の解決済みリンクを2回目のレンダリングへ運ぶ。各リンクは解析
// 結果から切り取られ、その開始位置にマーカー (ランダムな接頭辞を持つ通常の文字列) が書かれる。
// マーカーはテキストと同じようにレンダラー・サニタイザー・フラグメントパーサーを通るため、
// マーカーが出てくる場所が読み手にリンクが見える場所である。渡されたdocumentはその場で
// 書き換えられ、以降は再びレンダリングされない。
type wikilinkReplacer struct {
	prefix string
	links  []resolvedWikilink
}

// replaceWikilinkMatchesはpageLocationsが解決する一致をリンクにしてdocumentをもう一度
// レンダリングする。どれも解決しなければbodyHTMLをそのまま返す。sourceはdocumentの解析元で、
// matchesの位置が指すものである。
//
// マーカーのすべてが通常のテキストとして戻ってこない本文は、リンクなしで返す。走査とこの
// レンダリングの判断が食い違っているためで、マーカーを見せたりテキストを失ったりするよりは
// 記法をそのまま見せるほうがよい。
func replaceWikilinkMatches(
	source []byte,
	document ast.Node,
	bodyHTML string,
	matches []WikilinkMatch,
	spaceIdentifier model.SpaceIdentifier,
	pageLocations []PageLocation,
) string {
	var links []resolvedWikilink
	for _, match := range matches {
		if location := findPageLocation(match.Key, pageLocations); location != nil {
			links = append(links, resolvedWikilink{match: match, location: location})
		}
	}
	if len(links) == 0 {
		return bodyHTML
	}

	replacer := wikilinkReplacer{prefix: "Wikino" + rand.Text(), links: links}
	augmentedSource := replacer.mark(source, document)
	container, err := renderedTree(augmentedSource, document)
	if err != nil {
		slog.Warn("Wikiリンクのマーカーを含む本文のレンダリングに失敗", "error", err)

		return bodyHTML
	}

	if replaced := replacer.replace(container, false, spaceIdentifier); replaced != len(links) {
		slog.Warn("Wikiリンクのマーカーの一部が通常のテキストとして描画されなかった",
			"replaced", replaced, "links", len(links))

		return bodyHTML
	}

	return renderContainerChildren(container)
}

// markerはindexのリンクを表すマーカーを返す。
func (r *wikilinkReplacer) marker(index int) string {
	return r.prefix + strconv.Itoa(index) + "Z"
}

// linksOverlappingは [start, stop) と重なるリンクのインデックス範囲を返す。リンクは
// ソース順に並び互いに重ならない。ScanWikilinkMatchesが互いに素な区間からその順に読むためである。
func (r *wikilinkReplacer) linksOverlapping(start int, stop int) (int, int) {
	first := sort.Search(len(r.links), func(i int) bool { return r.links[i].match.Stop > start })
	last := first
	for last < len(r.links) && r.links[last].match.Start < stop {
		last++
	}

	return first, last
}

// markは解決済みのリンクをすべてdocumentから切り取り、リンクの開始位置にマーカーを書く。
// レンダラーが読むべきソースを返す。書き換えたraw HTMLブロックの行の分だけ長くなる。
//
// リンクはインライン内容のテキストノード、HTMLブロックの行、そして手前で開かれたraw text
// 要素がコードを読めるテキストに変えている場合はコードブロックの行とフェンスの情報文字列にある。
// ScanWikilinkMatchesはレンダリングがテキストとして見せるソースだけを読むため、リンク記法や
// raw HTMLのタグなどにリンクは無い。
func (r *wikilinkReplacer) mark(source []byte, document ast.Node) []byte {
	var texts []*ast.Text
	var lineBlocks []ast.Node
	var fences []*ast.FencedCodeBlock

	// 下のウォーカーは失敗しないため、ast.Walkが返すエラーはnilにしかならない。
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Text:
			texts = append(texts, n)
		case *ast.HTMLBlock:
			lineBlocks = append(lineBlocks, n)
		case *ast.CodeBlock:
			lineBlocks = append(lineBlocks, n)
		case *ast.FencedCodeBlock:
			lineBlocks = append(lineBlocks, n)
			if n.Info != nil {
				fences = append(fences, n)
			}
		}

		return ast.WalkContinue, nil
	})

	augmented := source
	for _, text := range texts {
		augmented = r.markText(augmented, text)
	}
	for _, fence := range fences {
		augmented = r.markFenceInfo(augmented, source, fence)
	}
	for _, block := range lineBlocks {
		augmented = r.markLines(augmented, source, block.Lines())
		if htmlBlock, ok := block.(*ast.HTMLBlock); ok && htmlBlock.HasClosure() {
			augmented, htmlBlock.ClosureLine = r.markLine(augmented, source, htmlBlock.ClosureLine)
		}
	}

	return augmented
}

// markTextは重なるリンクのバイトをnodeから取り除き、nodeの中で始まる各リンクの開始位置に
// マーカーを差し込む。リンクの手前のテキストは独立したノードになり、node自身は最後のリンクの
// 後ろを、持っているかもしれない改行とともに保持する。
//
// リンクが手前のノードで始まりここで終わることがある。強調は区切り文字でテキストを分ける一方、
// リンクはそれをまたいで続くためである。そうしたリンクのマーカーは既に書かれている。
//
// マーカーを文字列ノードではなくaugmentedを読むテキストノードにするのは、コードスパンの
// レンダラーが子をテキストノードとしてしか読まないためである。手前で開かれたraw text要素が
// コードスパンを読めるテキストに変えたとき、コードスパンはここへ到達する。
func (r *wikilinkReplacer) markText(augmented []byte, node *ast.Text) []byte {
	segment := node.Segment
	first, last := r.linksOverlapping(segment.Start, segment.Stop)
	if first == last {
		return augmented
	}

	parent := node.Parent()
	cursor := segment.Start
	padding := segment.Padding
	for i := first; i < last; i++ {
		link := r.links[i]
		if link.match.Start >= cursor {
			if link.match.Start > cursor || padding > 0 {
				before := ast.NewTextSegment(gmtext.NewSegmentPadding(cursor, link.match.Start, padding))
				before.SetRaw(node.IsRaw())
				parent.InsertBefore(parent, node, before)
			}
			var markerSegment gmtext.Segment
			augmented, markerSegment = appendMarker(augmented, r.marker(i))
			parent.InsertBefore(parent, node, ast.NewTextSegment(markerSegment))
			padding = 0
		}
		cursor = min(link.match.Stop, segment.Stop)
	}

	node.Segment = gmtext.NewSegmentPadding(cursor, segment.Stop, padding)

	return augmented
}

// markFenceInfoはリンクを含むフェンスの情報文字列を書き換える。ブロックは同じ行を持つ
// 新しいノードに差し替える。フェンス付きコードブロックは最初のレンダリングで読んだ言語を
// 覚えており、それをもう一度書いてしまうためである。
func (r *wikilinkReplacer) markFenceInfo(augmented []byte, source []byte, node *ast.FencedCodeBlock) []byte {
	marked := node.Info.Segment
	augmented, marked = r.markLine(augmented, source, marked)
	if marked == node.Info.Segment {
		return augmented
	}

	fresh := ast.NewFencedCodeBlock(ast.NewTextSegment(marked))
	fresh.SetLines(node.Lines())
	node.Parent().ReplaceChild(node.Parent(), node, fresh)

	return augmented
}

// markLinesはリンクを含むブロックの行を書き換える。レンダラーはそれらの行をソースから
// そのまま複製するためである。
func (r *wikilinkReplacer) markLines(augmented []byte, source []byte, lines *gmtext.Segments) []byte {
	for i := 0; i < lines.Len(); i++ {
		var marked gmtext.Segment
		augmented, marked = r.markLine(augmented, source, lines.At(i))
		lines.Set(i, marked)
	}

	return augmented
}

// markLineはリンクをマーカーに置き換えた行をaugmentedの末尾に書き、その行を指す
// セグメントを返す。リンクの無い行はそのまま返す。
func (r *wikilinkReplacer) markLine(augmented []byte, source []byte, segment gmtext.Segment) ([]byte, gmtext.Segment) {
	first, last := r.linksOverlapping(segment.Start, segment.Stop)
	if first == last {
		return augmented, segment
	}

	start := len(augmented)
	cursor := segment.Start
	for i := first; i < last; i++ {
		link := r.links[i]
		if link.match.Start >= cursor {
			augmented = append(augmented, source[cursor:link.match.Start]...)
			augmented = append(augmented, r.marker(i)...)
		}
		cursor = min(link.match.Stop, segment.Stop)
	}
	augmented = append(augmented, source[cursor:segment.Stop]...)

	return augmented, gmtext.NewSegmentPadding(start, len(augmented), segment.Padding)
}

// replaceはnode配下の通常のテキストノードにあるマーカーをリンク要素に変え、変えた数を
// 返す。skipElementsの要素の下にあるマーカーには触れない。走査はそこにリンクを報告しないため、
// 見つかれば両者の判断が食い違っている。
func (r *wikilinkReplacer) replace(node *html.Node, inSkip bool, spaceIdentifier model.SpaceIdentifier) int {
	inSkip = inSkip || (node.Type == html.ElementNode && skipElements[node.Data])
	if node.Type == html.TextNode {
		if inSkip || !strings.Contains(node.Data, r.prefix) {
			return 0
		}

		return r.replaceInText(node, spaceIdentifier)
	}

	replaced := 0
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		replaced += r.replace(child, inSkip, spaceIdentifier)
		child = next
	}

	return replaced
}

// replaceInTextはtextNodeをそれが持つ各マーカーの前後で分割し、マーカーの位置にそれが
// 表すリンク要素を置く。
func (r *wikilinkReplacer) replaceInText(textNode *html.Node, spaceIdentifier model.SpaceIdentifier) int {
	text := textNode.Data
	parent := textNode.Parent
	replaced, written, position := 0, 0, 0
	for {
		offset := strings.Index(text[position:], r.prefix)
		if offset < 0 {
			break
		}
		digitsStart := position + offset + len(r.prefix)
		digitsLength := strings.IndexByte(text[digitsStart:], 'Z')
		if digitsLength < 0 {
			break
		}
		position = digitsStart
		index, err := strconv.Atoi(text[digitsStart : digitsStart+digitsLength])
		if err != nil || index < 0 || index >= len(r.links) {
			continue
		}

		markerStart := digitsStart - len(r.prefix)
		if markerStart > written {
			parent.InsertBefore(&html.Node{Type: html.TextNode, Data: text[written:markerStart]}, textNode)
		}
		parent.InsertBefore(buildWikilinkNode(spaceIdentifier, r.links[index].location), textNode)
		replaced++
		position = digitsStart + digitsLength + 1
		written = position
	}
	if replaced == 0 {
		return 0
	}

	if written < len(text) {
		parent.InsertBefore(&html.Node{Type: html.TextNode, Data: text[written:]}, textNode)
	}
	parent.RemoveChild(textNode)

	return replaced
}
