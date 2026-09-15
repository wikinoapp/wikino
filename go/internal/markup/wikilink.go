package markup

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// wikilinkRegexはWikiリンク記法 [[...]] を検出する正規表現
var wikilinkRegex = regexp.MustCompile(`\[\[(.*?)\]\]`)

// wikilinkOpeningはWikiリンクを開く記法。これを一度も持たないソースはWikiリンクも持たない。
const wikilinkOpening = "[["

// skipElementsはWikiリンク変換をスキップするHTML要素のセット。
var skipElements = map[string]bool{
	"a":      true,
	"code":   true,
	"pre":    true,
	"script": true,
	"style":  true,
}

// WikilinkKeyはWikiリンクのトピック名とページタイトルのペア
type WikilinkKey struct {
	// RawはWikiリンクの原文 (例: "トピック名/ページタイトル")
	Raw string
	// TopicNameはトピック名
	TopicName string
	// PageTitleはページタイトル
	PageTitle string
}

// PageLocationはWikilinkKeyに対応する解決済みページ情報
type PageLocation struct {
	// Keyは元のWikiリンクキー
	Key WikilinkKey
	// TopicNameは解決されたトピック名
	TopicName string
	// PageIDはページID
	PageID model.PageID
	// PageNumberはページ番号 (URLで使用)
	PageNumber int
	// PageTitleはページタイトル (リンクテキストで使用)
	PageTitle string
}

// ScanWikilinksはMarkdown本文のWikiリンクのキーを現れる順に返す。[[ページ名]] 形式は
// currentTopicNameで補い、[[トピック名/ページ名]] 形式は分割する。ScanWikilinkMatchesと同じ
// リンクを読むため、コードとして書かれた [[...]]、既存のリンクの中の [[...]]、エスケープされた
// [[...]] はページを指さない。保存経路はキーが指すページを作成するので、画面がリンクとして
// 見せないものからページを作らないためである。
func ScanWikilinks(body string, currentTopicName string) []WikilinkKey {
	matches := ScanWikilinkMatches(body, currentTopicName)
	if len(matches) == 0 {
		return nil
	}

	keys := make([]WikilinkKey, 0, len(matches))
	for _, match := range matches {
		keys = append(keys, match.Key)
	}

	return keys
}

// ReplaceWikilinksはRenderMarkdownと同じようにbodyをレンダリングし、pageLocationsが
// 解決する本文のWikiリンクをそのページへのリンクにする。どの [[...]] がWikiリンクかは
// ScanWikilinkMatchesがソース上で決め、レンダリング済みHTMLから記法を探し直すことはしない。
// これにより表示とエクスポートは1つのリンク集合を読む。ページに解決できないリンクは書かれた
// まま残す。
func ReplaceWikilinks(body string, currentTopicName string, spaceIdentifier model.SpaceIdentifier, pageLocations []PageLocation) string {
	source, document, bodyHTML := renderBody(body)
	if document == nil {
		return bodyHTML
	}

	matches := ScanWikilinkMatches(string(source), currentTopicName)

	return replaceWikilinkMatches(source, document, bodyHTML, matches, spaceIdentifier, pageLocations)
}

// parseWikilinkRawはWikiリンクの原文からWikilinkKeyを構築する
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

// findPageLocationはWikilinkKeyに対応するPageLocationを検索する
func findPageLocation(key WikilinkKey, locations []PageLocation) *PageLocation {
	for i := range locations {
		if locations[i].Key.Raw == key.Raw {
			return &locations[i]
		}
	}
	return nil
}

// buildWikilinkNodeはWikiリンクの<a>要素ノードを構築する
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

// markdownWikilinkSyntaxRangesはMarkdownのリンクラベルの周囲の記法と、ラベルが属性に
// なる画像ノード全体を保護する。リンクラベルは描画後のツリーで判定するため、rawな終了タグや
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

// scanWikilinkSourceRangesは、レンダリングが [[...]] をWikiリンクとして扱わないソース
// 範囲を返す。既存のMarkdownリンク・画像、自動リンク、リンク参照定義、raw HTMLの構文そのもの、
// 中身が読み手にテキストとして届かないraw HTML要素、そしてscanRenderedWikilinkRangesが
// 描画後のツリーから読み取ったものを組み合わせる。
//
// 保護する要素は、ReplaceWikilinksがskipElementsで飛ばすものと、サニタイズポリシーが中身ごと
// 落とすものである。後者は画面に何も出ないため、その中に書かれた [[...]] もページのテキストでは
// ない。
//
// Markdownのコードを解析結果だけでなく描画後のツリーから保護するのは、手前で開かれたraw text
// 要素があると、ブロック・スパンがレンダリングされるcode要素がテキストとして読み手に届くため
// である。そのとき画面に出ているのは、表示側がほかと同じように変換する [[...]] である。
//
// リンク参照定義は何もレンダリングされないが、保護する範囲に含める。そこにあるのはその定義を
// 指すリンクのリンク先とタイトルであり、そこに書かれた [[...]] はテキストではなく構文だからである。
//
// 2つ目の戻り値は、中のテキストをMarkdownとして解析しない範囲、すなわちHTMLブロックと
// Markdownのコードである。その中のバックスラッシュはWikiリンクの角括弧が続いていても
// エスケープではなく通常の文字となる。
func scanWikilinkSourceRanges(source []byte) ([]byteRange, []byteRange) {
	document := md.Parser().Parse(gmtext.NewReader(source))

	// scanRenderedWikilinkRangesが文書にマーカーを付ける前に、Markdownとして解析しない
	// 範囲を読む。マーカーを付けるとコード・HTMLブロックのセグメントが拡張したソースへ移るため。
	literalRanges := scanMarkdownCodeRanges(document)
	linkRanges := scanMarkdownLinkRanges(source, document)
	var ranges []byteRange
	var rawHTMLRanges []byteRange
	var htmlBlockRanges []byteRange

	// 下のウォーカーは失敗しないため、ast.Walkが返すエラーはnilにしかならない。
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

// WikilinkMatchはMarkdown本文のWikiリンク1件と、本文中でそれが占めるバイト範囲。
type WikilinkMatch struct {
	// Keyはリンクが指すトピック名とページタイトル
	Key WikilinkKey

	// Startはリンクを開く "[[" の位置
	Start int

	// Stopはリンクを閉じる "]]" の直後の位置
	Stop int
}

// ScanWikilinkMatchesはMarkdown本文のWikiリンクを現れる順に返す。呼び出し元が置き換え
// られるよう、各リンクは本文中の位置を持つ。レンダリングが通常テキストとして扱うソース範囲だけを
// 走査する。そのためコード、サニタイズを要素として通過する既存のMarkdownリンク・画像、リンク参照
// 定義、raw HTMLのタグとコメント、raw HTMLのa・code・pre・script・style要素、そしてサニタイズが
// 中身ごと落とす要素は含めない。
//
// 角括弧はソースに書かれたまま読む。Markdownのテキスト内でエスケープされた開始括弧や、括弧の
// 代わりに置かれた文字参照はWikiリンクの構文にならない。HTMLブロックとMarkdownのコードの
// テキストではMarkdownのエスケープを解釈しない。
//
// これがWikiリンクの唯一の定義である。ReplaceWikilinksはこの一致を画面上のリンクにし、本文
// 自体を書き換える呼び出し元はソース上でこれを置き換える。
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

			// バックスラッシュは区間ではなく本文全体で数える。区間の直前に書かれた
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
