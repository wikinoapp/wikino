package markup

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// wikilinkRegex はWikiリンク記法 [[...]] を検出する正規表現
var wikilinkRegex = regexp.MustCompile(`\[\[(.*?)\]\]`)

// wikilinkOpening opens a wiki link. Source that never holds it holds no wiki link either.
//
// [Ja] wikilinkOpening は Wiki リンクを開く記法。これを一度も持たないソースは Wiki リンクも持たない。
const wikilinkOpening = "[["

// skipElements is the set of HTML elements where wiki-link conversion is skipped.
//
// [Ja] skipElements は Wiki リンク変換をスキップする HTML 要素のセット。
var skipElements = map[string]bool{
	"a":      true,
	"code":   true,
	"pre":    true,
	"script": true,
	"style":  true,
}

// WikilinkKey はWikiリンクのトピック名とページタイトルのペア
type WikilinkKey struct {
	// Raw はWikiリンクの原文（例: "トピック名/ページタイトル"）
	Raw string
	// TopicName はトピック名
	TopicName string
	// PageTitle はページタイトル
	PageTitle string
}

// PageLocation はWikilinkKeyに対応する解決済みページ情報
type PageLocation struct {
	// Key は元のWikiリンクキー
	Key WikilinkKey
	// TopicName は解決されたトピック名
	TopicName string
	// PageID はページID
	PageID model.PageID
	// PageNumber はページ番号（URLで使用）
	PageNumber int
	// PageTitle はページタイトル（リンクテキストで使用）
	PageTitle string
}

// ScanWikilinks はMarkdown本文からWikiリンクをパースし、
// トピック名とページタイトルのペアのリストを返す。
// [[ページ名]] 形式の場合はcurrentTopicNameを使用する。
// [[トピック名/ページ名]] 形式の場合はそのまま使用する。
func ScanWikilinks(body string, currentTopicName string) []WikilinkKey {
	matches := wikilinkRegex.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	var keys []WikilinkKey
	for _, match := range matches {
		raw := strings.TrimSpace(match[1])
		if raw == "" {
			continue
		}

		parts := strings.SplitN(raw, "/", 2)
		if len(parts) == 2 {
			// [[トピック名/ページ名]] 形式
			keys = append(keys, WikilinkKey{
				Raw:       raw,
				TopicName: parts[0],
				PageTitle: parts[1],
			})
		} else {
			// [[ページ名]] 形式 — 現在のトピック名を使用
			keys = append(keys, WikilinkKey{
				Raw:       raw,
				TopicName: currentTopicName,
				PageTitle: parts[0],
			})
		}
	}

	return keys
}

// ReplaceWikilinks はbodyHTML内のWikiリンクをHTML <a>タグに変換する。
// <a>, <code>, <pre>, <script>, <style>タグ内のWikiリンクは変換しない。
// ページが存在する場合は <a href="/s/{spaceIdentifier}/pages/{pageNumber}">ページタイトル</a> に変換し、
// 存在しない場合はプレーンテキストのまま残す。
func ReplaceWikilinks(bodyHTML string, currentTopicName string, spaceIdentifier model.SpaceIdentifier, pageLocations []PageLocation) string {
	if !strings.Contains(bodyHTML, "[[") {
		return bodyHTML
	}

	container, err := parseHTMLFragmentWithContainer(bodyHTML)
	if err != nil {
		slog.Warn("Wikiリンク変換時のHTMLパースに失敗", "error", err)
		return bodyHTML
	}

	modified := processWikilinkNodes(container, currentTopicName, spaceIdentifier, pageLocations, false)

	if !modified {
		return bodyHTML
	}

	return renderContainerChildren(container)
}

// processWikilinkNodes はDOMツリーを再帰的に走査し、
// テキストノード内のWikiリンクを<a>要素に変換する。変更があればtrueを返す。
func processWikilinkNodes(n *html.Node, currentTopicName string, spaceIdentifier model.SpaceIdentifier, pageLocations []PageLocation, inSkip bool) bool {
	if n.Type == html.ElementNode && skipElements[n.Data] {
		inSkip = true
	}

	if n.Type == html.TextNode && !inSkip && strings.Contains(n.Data, "[[") {
		return replaceWikilinksInTextNode(n, currentTopicName, spaceIdentifier, pageLocations)
	}

	modified := false
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if processWikilinkNodes(c, currentTopicName, spaceIdentifier, pageLocations, inSkip) {
			modified = true
		}
		c = next
	}

	return modified
}

// replaceWikilinksInTextNode はテキストノード内のWikiリンクを検出して<a>要素に置換する。
// テキストを分割し、Wikiリンク部分を<a>ノードに、それ以外をテキストノードに変換する。
func replaceWikilinksInTextNode(textNode *html.Node, currentTopicName string, spaceIdentifier model.SpaceIdentifier, pageLocations []PageLocation) bool {
	text := textNode.Data
	matches := wikilinkRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return false
	}

	// 置換対象があるかチェック
	hasReplacement := false
	for _, loc := range matches {
		raw := strings.TrimSpace(text[loc[2]:loc[3]])
		if raw == "" {
			continue
		}
		key := parseWikilinkRaw(raw, currentTopicName)
		if findPageLocation(key, pageLocations) != nil {
			hasReplacement = true
			break
		}
	}

	if !hasReplacement {
		return false
	}

	parent := textNode.Parent
	lastEnd := 0

	for _, loc := range matches {
		raw := strings.TrimSpace(text[loc[2]:loc[3]])
		if raw == "" {
			continue
		}

		key := parseWikilinkRaw(raw, currentTopicName)
		pl := findPageLocation(key, pageLocations)
		if pl == nil {
			continue
		}

		// マッチ前のテキストを挿入
		if loc[0] > lastEnd {
			before := &html.Node{
				Type: html.TextNode,
				Data: text[lastEnd:loc[0]],
			}
			parent.InsertBefore(before, textNode)
		}

		// <a>要素を挿入
		aNode := buildWikilinkNode(spaceIdentifier, pl)
		parent.InsertBefore(aNode, textNode)

		lastEnd = loc[1]
	}

	// 残りのテキスト
	if lastEnd < len(text) {
		after := &html.Node{
			Type: html.TextNode,
			Data: text[lastEnd:],
		}
		parent.InsertBefore(after, textNode)
	}

	// 元のテキストノードを削除
	parent.RemoveChild(textNode)

	return true
}

// parseWikilinkRaw はWikiリンクの原文からWikilinkKeyを構築する
func parseWikilinkRaw(raw string, currentTopicName string) WikilinkKey {
	parts := strings.SplitN(raw, "/", 2)
	if len(parts) == 2 {
		return WikilinkKey{
			Raw:       raw,
			TopicName: parts[0],
			PageTitle: parts[1],
		}
	}
	return WikilinkKey{
		Raw:       raw,
		TopicName: currentTopicName,
		PageTitle: parts[0],
	}
}

// findPageLocation はWikilinkKeyに対応するPageLocationを検索する
func findPageLocation(key WikilinkKey, locations []PageLocation) *PageLocation {
	for i := range locations {
		if locations[i].Key.Raw == key.Raw {
			return &locations[i]
		}
	}
	return nil
}

// buildWikilinkNode はWikiリンクの<a>要素ノードを構築する
func buildWikilinkNode(spaceIdentifier model.SpaceIdentifier, pl *PageLocation) *html.Node {
	href := fmt.Sprintf("/s/%s/pages/%d", url.PathEscape(string(spaceIdentifier)), pl.PageNumber)

	aNode := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.A,
		Data:     "a",
		Attr: []html.Attribute{
			{Key: "href", Val: href},
		},
	}

	linkText := &html.Node{
		Type: html.TextNode,
		Data: pl.PageTitle,
	}
	aNode.AppendChild(linkText)

	return aNode
}

// markdownWikilinkSyntaxRanges protects the notation around a Markdown link label and the whole
// image node, whose label becomes an attribute. Link labels are classified using the rendered
// tree, so raw end tags and sanitizer decisions affect them exactly as they do on the screen.
//
// [Ja] markdownWikilinkSyntaxRanges は Markdown のリンクラベルの周囲の記法と、ラベルが属性に
// なる画像ノード全体を保護する。リンクラベルは描画後のツリーで判定するため、raw な終了タグや
// サニタイザーの判断が画面と同じように反映される。
func markdownWikilinkSyntaxRanges(node ast.Node, ranges markdownLinkSourceRanges) []byteRange {
	if node.Kind() == ast.KindImage {
		return []byteRange{ranges.node}
	}
	return []byteRange{
		{start: ranges.node.start, stop: ranges.label.start},
		{start: ranges.label.stop, stop: ranges.node.stop},
	}
}

// scanWikilinkSourceRanges returns the source ranges where rendering does not treat [[...]] as
// a wiki link. It combines existing Markdown links and images, autolinks, link reference
// definitions, the syntax of raw HTML itself, the raw HTML elements whose content never reaches
// the reader as text, and what scanRenderedWikilinkRanges reads out of the rendered tree.
//
// The protected elements are those ReplaceWikilinks skips through skipElements, plus those the
// sanitization policy drops together with their content. The screen shows nothing of the latter,
// so a [[...]] written inside one is not text of the page either.
//
// Markdown code is protected through the rendered tree rather than from the parse alone, because a
// raw-text element opened earlier makes the code element the block or span renders to reach the
// reader as text. What the screen then shows is a [[...]] that display converts like any other.
//
// A link reference definition is part of the protected set even though nothing of it is rendered:
// what it holds is the destination and the title of the links that name it, so a [[...]] written
// there is syntax rather than text.
//
// The second result holds the ranges whose text is not parsed as Markdown: HTML blocks and
// Markdown code. Backslashes there are literal bytes rather than escapes, even when followed by
// wiki-link brackets.
//
// [Ja] scanWikilinkSourceRanges は、レンダリングが [[...]] を Wiki リンクとして扱わないソース
// 範囲を返す。既存の Markdown リンク・画像、自動リンク、リンク参照定義、raw HTML の構文そのもの、
// 中身が読み手にテキストとして届かない raw HTML 要素、そして scanRenderedWikilinkRanges が
// 描画後のツリーから読み取ったものを組み合わせる。
//
// 保護する要素は、ReplaceWikilinks が skipElements で飛ばすものと、サニタイズポリシーが中身ごと
// 落とすものである。後者は画面に何も出ないため、その中に書かれた [[...]] もページのテキストでは
// ない。
//
// Markdown のコードを解析結果だけでなく描画後のツリーから保護するのは、手前で開かれた raw text
// 要素があると、ブロック・スパンがレンダリングされる code 要素がテキストとして読み手に届くため
// である。そのとき画面に出ているのは、表示側がほかと同じように変換する [[...]] である。
//
// リンク参照定義は何もレンダリングされないが、保護する範囲に含める。そこにあるのはその定義を
// 指すリンクのリンク先とタイトルであり、そこに書かれた [[...]] はテキストではなく構文だからである。
//
// 2 つ目の戻り値は、中のテキストを Markdown として解析しない範囲、すなわち HTML ブロックと
// Markdown のコードである。その中のバックスラッシュは Wiki リンクの角括弧が続いていても
// エスケープではなく通常の文字となる。
func scanWikilinkSourceRanges(source []byte) ([]byteRange, []byteRange) {
	document := md.Parser().Parse(gmtext.NewReader(source))

	// The literal ranges are read before scanRenderedWikilinkRanges marks the document, since
	// marking moves the segments of the code and HTML block nodes onto the augmented source.
	//
	// [Ja] scanRenderedWikilinkRanges が文書にマーカーを付ける前に、Markdown として解析しない
	// 範囲を読む。マーカーを付けるとコード・HTML ブロックのセグメントが拡張したソースへ移るため。
	literalRanges := scanMarkdownCodeRanges(document)
	linkRanges := scanMarkdownLinkRanges(source, document)
	var ranges []byteRange
	var rawHTMLRanges []byteRange
	var htmlBlockRanges []byteRange

	// The walker below never fails, so the error ast.Walk returns can only be nil.
	//
	// [Ja] 下のウォーカーは失敗しないため、ast.Walk が返すエラーは nil にしかならない。
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node.Kind() {
		case ast.KindLink, ast.KindImage:
			sourceRanges, ok := linkRanges[node]
			if !ok {
				if node.Kind() == ast.KindImage {
					return ast.WalkSkipChildren, nil
				}
				break
			}

			ranges = append(ranges, markdownWikilinkSyntaxRanges(node, sourceRanges)...)
			if node.Kind() == ast.KindImage {
				return ast.WalkSkipChildren, nil
			}
		case ast.KindAutoLink:
			autoLink, ok := node.(*ast.AutoLink)
			if !ok || node.Pos() < 0 {
				break
			}

			start := node.Pos()
			stop := start + len(autoLink.Label(source))
			if start < len(source) && source[start] == '<' {
				stop += 2
			}
			ranges = append(ranges, byteRange{start: start, stop: stop})
		case ast.KindLinkReferenceDefinition:
			lines := node.Lines()
			for i := 0; i < lines.Len(); i++ {
				segment := lines.At(i)
				ranges = append(ranges, byteRange{start: segment.Start, stop: segment.Stop})
			}
		case ast.KindHTMLBlock, ast.KindRawHTML:
			if sourceRange, ok := rawHTMLNodeRange(node); ok {
				rawHTMLRanges = append(rawHTMLRanges, sourceRange)
				if node.Kind() == ast.KindHTMLBlock {
					htmlBlockRanges = append(htmlBlockRanges, sourceRange)
				}
			}
		}

		return ast.WalkContinue, nil
	})

	elementRanges, tokenRanges := scanRawHTMLRanges(source, rawHTMLScanOptions{
		rawHTMLRanges:  rawHTMLRanges,
		protectElement: func(_ html.TokenType, name string, _ []byte) bool { return rawTextDiscardedElements[name] },
		discardElement: func(name string) bool { return contentDiscardedElements[name] },
		acceptToken: func(tokenType html.TokenType, _ string) bool {
			return tokenType != html.TextToken
		},
	})

	ranges = append(ranges, elementRanges...)
	ranges = append(ranges, tokenRanges...)
	ranges = append(ranges, scanRenderedWikilinkRanges(source, document)...)

	return normalizeByteRanges(ranges), normalizeByteRanges(append(literalRanges, htmlBlockRanges...))
}

// WikilinkMatch is one wiki link of a Markdown body, together with the bytes it occupies there.
//
// [Ja] WikilinkMatch は Markdown 本文の Wiki リンク 1 件と、本文中でそれが占めるバイト範囲。
type WikilinkMatch struct {
	// Key is the topic name and page title the link names.
	//
	// [Ja] Key はリンクが指すトピック名とページタイトル
	Key WikilinkKey

	// Start is the offset of the "[[" that opens the link.
	//
	// [Ja] Start はリンクを開く "[[" の位置
	Start int

	// Stop is the offset just past the "]]" that closes it.
	//
	// [Ja] Stop はリンクを閉じる "]]" の直後の位置
	Stop int
}

// ScanWikilinkMatches returns the wiki links of a Markdown body in the order they appear, each one
// carrying where it sits in the body so that a caller can replace it. It scans only source ranges
// that rendering treats as ordinary text. Code, existing Markdown links and images whose element
// survives sanitization, link reference definitions, the tags and comments of raw HTML, the raw
// HTML a, code, pre, script and style elements, and the elements sanitization drops together with
// their content are therefore left out.
//
// Each of those ranges is scanned on its own, since a wiki link has to open and close inside one of
// them to be one on the screen: rendering hands ReplaceWikilinks the text of a body already split
// at every piece of syntax between them.
//
// The brackets are read from the source as they are written. An escaped opening bracket in
// Markdown text or a character reference standing in for a bracket does not form wiki-link
// syntax. The text of an HTML block and of Markdown code does not interpret Markdown escapes.
//
// This applies to the Markdown source the boundary the screen shows, for callers that rewrite the
// body itself.
//
// [Ja] ScanWikilinkMatches は Markdown 本文の Wiki リンクを現れる順に返す。呼び出し元が置き換え
// られるよう、各リンクは本文中の位置を持つ。レンダリングが通常テキストとして扱うソース範囲だけを
// 走査する。そのためコード、サニタイズを要素として通過する既存の Markdown リンク・画像、リンク参照
// 定義、raw HTML のタグとコメント、raw HTML の a・code・pre・script・style 要素、そしてサニタイズが
// 中身ごと落とす要素は含めない。
//
// 角括弧はソースに書かれたまま読む。Markdown のテキスト内でエスケープされた開始括弧や、括弧の
// 代わりに置かれた文字参照は Wiki リンクの構文にならない。HTML ブロックと Markdown のコードの
// テキストでは Markdown のエスケープを解釈しない。
//
// これは画面が見せている境界を、本文自体を書き換える呼び出し元のために Markdown のソースへ
// 適用するものである。
func ScanWikilinkMatches(body string, currentTopicName string) []WikilinkMatch {
	if !strings.Contains(body, wikilinkOpening) {
		return nil
	}

	protectedRanges, literalRanges := scanWikilinkSourceRanges([]byte(body))

	var matches []WikilinkMatch
	literalRangeCursor := byteRangeCursor{ranges: literalRanges}

	for _, span := range unprotectedSpans(len(body), protectedRanges) {
		for _, loc := range wikilinkRegex.FindAllStringSubmatchIndex(body[span.start:span.stop], -1) {
			start := span.start + loc[0]

			// The backslashes are counted in the whole body rather than in the span: an escape
			// written right before a span sits in the range that ends where the span begins.
			//
			// [Ja] バックスラッシュは区間ではなく本文全体で数える。区間の直前に書かれた
			// エスケープは、その区間が始まる位置で終わる範囲の中にあるためである。
			if !literalRangeCursor.overlaps(start, start+1) {
				slashStart := start
				for slashStart > 0 && body[slashStart-1] == '\\' {
					slashStart--
				}
				if (start-slashStart)%2 != 0 {
					continue
				}
			}

			raw := strings.TrimSpace(body[span.start+loc[2] : span.start+loc[3]])
			if raw == "" {
				continue
			}

			matches = append(matches, WikilinkMatch{
				Key:   parseWikilinkRaw(raw, currentTopicName),
				Start: start,
				Stop:  span.start + loc[1],
			})
		}
	}

	return matches
}
