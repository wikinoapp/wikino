package markup

import (
	"bytes"
	"cmp"
	"slices"
	"sort"

	"github.com/yuin/goldmark/ast"
	"golang.org/x/net/html"
)

// byteRangeはMarkdown本文へのバイト位置の半開区間。
type byteRange struct {
	start int
	stop  int
}

// normalizeByteRangesは範囲を開始位置順に並べ、重なる範囲と接する範囲を統合する。
// ソース順に位置を照合する呼び出し元は、その結果を1つのカーソルで前へ進められる。
func normalizeByteRanges(ranges []byteRange) []byteRange {
	if len(ranges) < 2 {
		return ranges
	}

	slices.SortFunc(ranges, func(a, b byteRange) int {
		if byStart := cmp.Compare(a.start, b.start); byStart != 0 {
			return byStart
		}
		return cmp.Compare(a.stop, b.stop)
	})

	merged := ranges[:1]
	for _, current := range ranges[1:] {
		last := &merged[len(merged)-1]
		if current.start <= last.stop {
			last.stop = max(last.stop, current.stop)
			continue
		}
		merged = append(merged, current)
	}

	return merged
}

// byteRangeCursorはソース順に単調増加して届く範囲を、正規化済みの範囲集合と照合する。
// 一連の照合全体で、それぞれの保護範囲を通過するのは高々1回である。
type byteRangeCursor struct {
	ranges []byteRange
	index  int
}

// overlapsは [start, stop) がカーソルの現在位置以降の範囲と重なるかを返す。
func (cursor *byteRangeCursor) overlaps(start int, stop int) bool {
	for cursor.index < len(cursor.ranges) && cursor.ranges[cursor.index].stop <= start {
		cursor.index++
	}

	return cursor.index < len(cursor.ranges) &&
		start < cursor.ranges[cursor.index].stop &&
		cursor.ranges[cursor.index].start < stop
}

// containsは [start, stop) の全体をカーソルのいずれかの範囲が含むかを返す。正規化済みの
// 範囲集合には重なる範囲も接する範囲も無いため、いずれかの範囲を越えて伸びる区間はどの範囲にも
// 含まれない。
func (cursor *byteRangeCursor) contains(start int, stop int) bool {
	for cursor.index < len(cursor.ranges) && cursor.ranges[cursor.index].stop <= start {
		cursor.index++
	}

	return cursor.index < len(cursor.ranges) &&
		cursor.ranges[cursor.index].start <= start &&
		stop <= cursor.ranges[cursor.index].stop
}

// unprotectedSpansは [0, length) のうちrangesが覆っていない部分を、ソース順に返す。
// rangesは正規化済みである必要がある。
//
// これらの部分の内側に収まる必要があるものを探す呼び出し元は、ソース全体ではなく各区間へ照合を
// 行う。レンダリングが本文のテキストに対して行っているのがこれで、保護される構文がテキストを
// 別々のまとまりへ分割し、それぞれが独立に読まれる。ソース全体へ照合すると、保護範囲の手前から
// 始まり後ろで終わる一致が、画面がまとまりの中に見つける一致を覆って隠してしまう。
func unprotectedSpans(length int, ranges []byteRange) []byteRange {
	var spans []byteRange
	start := 0

	for _, protected := range ranges {
		if start >= length {
			return spans
		}
		if protected.start > start {
			spans = append(spans, byteRange{start: start, stop: min(protected.start, length)})
		}
		start = max(start, protected.stop)
	}

	if start >= length {
		return spans
	}

	return append(spans, byteRange{start: start, stop: length})
}

// byteRangesHoldは、rangesのいずれかがpositionを含むかを返す。rangesは正規化済みである
// 必要があり、これにより照合する位置がソース順に届かない呼び出し元も、カーソルを前へ進めるのでは
// なく探索できる。
func byteRangesHold(ranges []byteRange, position int) bool {
	index := sort.Search(len(ranges), func(i int) bool { return ranges[i].stop > position })

	return index < len(ranges) && ranges[index].start <= position
}

// scanMarkdownCodeRangesはsourceのうちMarkdownの構文がコードとしている範囲を返す。
// フェンス付きコードブロックの情報文字列と本文各行または字下げコードブロックの各行、そして
// コードスパンの中身である。documentは呼び出し元が既に持っている解析結果で、レンダリングが使う
// パーサーが作ったものなので、本文の読み方は画面上と一致する。
//
// raw HTMLのcode要素・pre要素はscanRenderedWikilinkRangesが扱い、実際のHTMLツリーから
// それらが囲むテキストを判定する。
func scanMarkdownCodeRanges(document ast.Node) []byteRange {
	var ranges []byteRange
	// 下のウォーカーは失敗しないため、ast.Walkが返すエラーはnilにしかならない。
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node.Kind() {
		case ast.KindFencedCodeBlock:
			fencedCodeBlock, ok := node.(*ast.FencedCodeBlock)
			if ok && fencedCodeBlock.Info != nil {
				segment := fencedCodeBlock.Info.Segment
				ranges = append(ranges, byteRange{start: segment.Start, stop: segment.Stop})
			}

			lines := node.Lines()
			for i := 0; i < lines.Len(); i++ {
				segment := lines.At(i)
				ranges = append(ranges, byteRange{start: segment.Start, stop: segment.Stop})
			}
		case ast.KindCodeBlock:
			lines := node.Lines()
			for i := 0; i < lines.Len(); i++ {
				segment := lines.At(i)
				ranges = append(ranges, byteRange{start: segment.Start, stop: segment.Stop})
			}
		case ast.KindCodeSpan:
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if textNode, ok := child.(*ast.Text); ok {
					ranges = append(
						ranges,
						byteRange{start: textNode.Segment.Start, stop: textNode.Segment.Stop},
					)
				}
			}
		}

		return ast.WalkContinue, nil
	})

	return ranges
}

// markdownLinkは本パッケージがast.Link・ast.Imageから読む部分。goldmarkは2つのノードに
// 同じフィールドを持たせているが、それらへ到達するための共通の型は用意していない。
type markdownLink struct {
	// referenceはリンクが指す参照で、リンクが自身のインラインのリンク先を持つ場合はnil
	reference *ast.ReferenceLink

	// destinationはパーサーが読んだリンク先
	destination []byte
}

// markdownLinkOfはnodeがMarkdownのリンク・画像であれば、その持ち物を返す。
func markdownLinkOf(node ast.Node) (markdownLink, bool) {
	switch typedNode := node.(type) {
	case *ast.Link:
		return markdownLink{reference: typedNode.Reference, destination: typedNode.Destination}, true
	case *ast.Image:
		return markdownLink{reference: typedNode.Reference, destination: typedNode.Destination}, true
	default:
		return markdownLink{}, false
	}
}

// markdownLinkSourceRangesは構文全体、それが囲むラベル、置換可能なリンク先の範囲を分けて
// 保持する。参照リンク自身のソース範囲にはリンク先がない。
//
// ラベルを残りと分けて持つのは、レンダリングが要素を落としつつラベルの中身は残すことがあり、
// その中身が読み手に通常のテキストとして届くためである。
type markdownLinkSourceRanges struct {
	node           byteRange
	label          byteRange
	destination    byteRange
	hasDestination bool
}

// scanMarkdownLinkRangesは外側のリンクより先に子の範囲を計算する。raw HTML、コードスパン、
// 入れ子のリンク・画像全体は、外側ラベルの角括弧の走査では読み飛ばす。各子の範囲で収集済みの
// 子孫を置き換えるため、無関係な範囲をラベルごとにコピーすることはない。
func scanMarkdownLinkRanges(source []byte, document ast.Node) map[ast.Node]markdownLinkSourceRanges {
	ranges := make(map[ast.Node]markdownLinkSourceRanges)
	var labelStarts []int
	var syntaxRanges []byteRange

	// ウォーカーは失敗しない。リンクから退出する時点では、子をすべて走査済みである。
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if node.Kind() == ast.KindLink || node.Kind() == ast.KindImage {
			if entering {
				labelStarts = append(labelStarts, len(syntaxRanges))
				return ast.WalkContinue, nil
			}

			start := labelStarts[len(labelStarts)-1]
			labelStarts = labelStarts[:len(labelStarts)-1]
			labelSyntax := syntaxRanges[start:]
			nodeRange, ok := markdownLinkNodeRange(source, node, labelSyntax)
			if ok {
				destination, found := markdownLinkDestinationRange(source, node, labelSyntax)
				label, _ := markdownLinkLabelContentRange(source, node, labelSyntax)
				ranges[node] = markdownLinkSourceRanges{
					node: nodeRange, label: label, destination: destination, hasDestination: found,
				}
			}
			syntaxRanges = syntaxRanges[:start]
			if ok && len(labelStarts) > 0 {
				syntaxRanges = append(syntaxRanges, nodeRange)
			}
			return ast.WalkContinue, nil
		}
		if !entering || len(labelStarts) == 0 {
			return ast.WalkContinue, nil
		}

		switch node.Kind() {
		case ast.KindRawHTML:
			if sourceRange, ok := rawHTMLNodeRange(node); ok {
				syntaxRanges = append(syntaxRanges, sourceRange)
			}
		case ast.KindCodeSpan:
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if textNode, ok := child.(*ast.Text); ok {
					syntaxRanges = append(syntaxRanges, byteRange{
						start: textNode.Segment.Start, stop: textNode.Segment.Stop,
					})
				}
			}
		}
		return ast.WalkContinue, nil
	})

	return ranges
}

// markdownLinkNodeRangeはMarkdownのリンク・画像ノードがソース中で占める範囲を返す。
// goldmarkが保持するのは開始位置だけなので、対応の取れたラベルをたどり、その後ろはリンクの形が
// 示すものをたどる。丸括弧に囲まれたインラインのリンク先か、full・collapsedの参照のラベルか、
// あるいは何も続かないかである。
//
// ショートカット参照はラベルで終わる。続く "[...]" をその一部として読むと、"[a][[ページ]]" の
// Wikiリンクのような、パーサーがリンクの外に残したテキストまで飲み込んでしまう。
//
// labelSyntaxRangesはラベルを区切らない角括弧を持つ解析済み構文の範囲である。必要とするのは
// ラベルだけで、インラインのリンク先と参照のラベルは生のまま読まれる。
func markdownLinkNodeRange(source []byte, node ast.Node, labelSyntaxRanges []byteRange) (byteRange, bool) {
	link, ok := markdownLinkOf(node)
	if !ok {
		return byteRange{}, false
	}

	start, stop, ok := markdownLinkLabelRange(source, node, labelSyntaxRanges)
	if !ok {
		return byteRange{}, false
	}

	switch {
	case link.reference == nil:
		if stop < len(source) && source[stop] == '(' {
			// 解析済みリンク先の範囲を使い、その中の丸括弧でリンクを区切らないようにする。
			// 開き丸括弧とリンク先の間に引用・リストの接頭辞がある場合も同じである。
			var destinationRanges []byteRange
			if destination, found := destinationRangeAt(source, node, stop+1, link.destination); found {
				destinationRanges = []byteRange{destination}
			}
			if destinationStop, found := scanBalancedDelimiter(source, stop, '(', ')', true, destinationRanges); found {
				stop = destinationStop
			}
		}
	case link.reference.Type != ast.ReferenceLinkShortcut:
		if stop < len(source) && source[stop] == '[' {
			if referenceStop, found := scanBalancedDelimiter(source, stop, '[', ']', false, nil); found {
				stop = referenceStop
			}
		}
	}

	return byteRange{start: start, stop: stop}, true
}

// markdownLinkDestinationRangeはMarkdownのリンク・画像のリンク先がソース中で占める範囲を
// 返す。周囲の構文と、後ろに続くタイトルは含めない。参照リンクでは何も返さない。そのリンク先は
// リンク自身ではなくリンク参照定義に書かれており、linkReferenceDefinitionDestinationRangeが
// そちらから返すためである。
func markdownLinkDestinationRange(source []byte, node ast.Node, labelSyntaxRanges []byteRange) (byteRange, bool) {
	link, ok := markdownLinkOf(node)
	if !ok || link.reference != nil {
		return byteRange{}, false
	}

	_, labelStop, ok := markdownLinkLabelRange(source, node, labelSyntaxRanges)
	if !ok || labelStop >= len(source) || source[labelStop] != '(' {
		return byteRange{}, false
	}

	return destinationRangeAt(source, node, labelStop+1, link.destination)
}

// linkReferenceDefinitionDestinationRangeは、リンク参照定義のリンク先がソース中で占める範囲
// を返す。周囲の構文と、後ろに続くタイトルは含めない。
//
// 参照リンクのリンク先が書かれているのはここであり、その形のリンクについて本文の指す先を書き換える
// 呼び出し元が触れられる唯一の場所である。リンク自身は置き換えるリンク先を持たず、1つの定義を
// 複数のリンクが指すこともある。
func linkReferenceDefinitionDestinationRange(source []byte, node ast.Node) (byteRange, bool) {
	definition, ok := node.(*ast.LinkReferenceDefinition)
	if !ok {
		return byteRange{}, false
	}

	labelStart := node.Pos()
	if labelStart < 0 || labelStart >= len(source) {
		return byteRange{}, false
	}

	for labelStart < len(source) && isHTMLSpace(source[labelStart]) {
		labelStart++
	}
	if labelStart >= len(source) || source[labelStart] != '[' {
		return byteRange{}, false
	}

	// リンク参照定義のラベルは生のまま読まれるため、その中でコードスパンは解決されず、
	// そこにある "]" は何に囲まれていてもラベルを閉じる。
	labelStop, ok := scanBalancedDelimiter(source, labelStart, '[', ']', false, nil)
	if !ok || labelStop >= len(source) || source[labelStop] != ':' {
		return byteRange{}, false
	}

	return destinationRangeAt(source, node, labelStop+1, definition.Destination)
}

// destinationRangeAtは、start以降で最初に空白でないバイトから始まるリンク先がソース中で
// 占める範囲を返す。リンク先は裸で書かれていても山括弧で囲まれていてもよい。ブロックの行セグメント
// はパーサーへの入力と同じく引用・リストの接頭辞を除いている。リンク先は1セグメント内に収まり、
// その前の空白だけが行をまたぐ。
//
// 範囲を返すのは、それがパーサーの読んだバイト列であるdestinationを覆っている場合だけである。
// そうでなければ、ここで求めた位置はリンク先とは別のものを指しており、置き換える呼び出し元は
// 誤ったバイトを上書きすることになる。
func destinationRangeAt(source []byte, node ast.Node, start int, destination []byte) (byteRange, bool) {
	for node != nil && node.Type() == ast.TypeInline {
		node = node.Parent()
	}
	if node == nil {
		return byteRange{}, false
	}
	lines := node.Lines()
	lineIndex := sort.Search(lines.Len(), func(i int) bool { return lines.At(i).Stop > start })
	for ; lineIndex < lines.Len(); lineIndex++ {
		line := lines.At(lineIndex)
		start = max(start, line.Start)
		for start < line.Stop && isHTMLSpace(source[start]) {
			start++
		}
		if start == line.Stop {
			continue
		}
		if source[start] == '<' {
			start++
		}
		stop := start + len(destination)
		if stop > line.Stop || !bytes.Equal(source[start:stop], destination) {
			return byteRange{}, false
		}
		return byteRange{start: start, stop: stop}, true
	}
	return byteRange{}, false
}

// markdownLinkLabelRangeはMarkdownのリンク・画像ノードの開始位置と、そのラベルを閉じる
// "]" の直後の位置を返す。labelSyntaxRangesはラベル内の解析済み構文で、その中の角括弧では
// ラベルは開かず閉じない。
func markdownLinkLabelRange(source []byte, node ast.Node, labelSyntaxRanges []byteRange) (int, int, bool) {
	start := node.Pos()
	if start < 0 || start >= len(source) {
		return 0, 0, false
	}

	labelStart := start
	if source[labelStart] == '!' {
		labelStart++
	}
	if labelStart >= len(source) || source[labelStart] != '[' {
		return 0, 0, false
	}

	stop, ok := scanBalancedDelimiter(source, labelStart, '[', ']', false, labelSyntaxRanges)
	if !ok {
		return 0, 0, false
	}

	return start, stop, true
}

// markdownLinkLabelContentRangeは、Markdownのリンク・画像のラベルの角括弧に挟まれた範囲を
// 返す。角括弧自体は含めない。サニタイズはリンクがレンダリングされた要素を落としつつこの中身は
// 残すことがあり、その場合は読み手に通常のテキストとして届く。
func markdownLinkLabelContentRange(source []byte, node ast.Node, labelSyntaxRanges []byteRange) (byteRange, bool) {
	start, stop, ok := markdownLinkLabelRange(source, node, labelSyntaxRanges)
	if !ok {
		return byteRange{}, false
	}

	if source[start] == '!' {
		start++
	}

	return byteRange{start: start + 1, stop: stop - 1}, true
}

// scanBalancedDelimiterはsource[start] と対になる閉じ区切りの直後の位置を返す。
// バックスラッシュによるエスケープを飛ばし、必要に応じて引用符内の区切りも無視する。skipRangesの
// いずれかの中にあるバイトは区切りにならない。skipRangesは正規化済みである必要があり、この走査で
// 中身を解釈しない解析済み構文やリンク先を保持する。
func scanBalancedDelimiter(
	source []byte,
	start int,
	opening byte,
	closing byte,
	ignoreQuoted bool,
	skipRanges []byteRange,
) (int, bool) {
	depth := 0
	var quote byte
	angleDestinationStart := -1
	inAngleDestination := false

	// 山括弧付きリンク先の中の丸括弧は、外側のインラインリンクの区切りではなくリンク先の
	// 通常のバイトである。その形を開始できるのは、先頭の空白を除いた最初のバイトだけである。
	if ignoreQuoted && opening == '(' {
		angleDestinationStart = start + 1
		for angleDestinationStart < len(source) && isHTMLSpace(source[angleDestinationStart]) {
			angleDestinationStart++
		}
		if angleDestinationStart >= len(source) || source[angleDestinationStart] != '<' {
			angleDestinationStart = -1
		}
	}

	// 飛ばす範囲には、startを含みうる最初のものから入る。そうした範囲の多い本文で、走査の
	// たびに集合の先頭からたどることにならないようにするためである。
	skipIndex := sort.Search(len(skipRanges), func(i int) bool {
		return skipRanges[i].stop > start
	})

	for i := start; i < len(source); i++ {
		for skipIndex < len(skipRanges) && skipRanges[skipIndex].stop <= i {
			skipIndex++
		}
		if skipIndex < len(skipRanges) && skipRanges[skipIndex].start <= i {
			i = skipRanges[skipIndex].stop - 1
			continue
		}

		current := source[i]
		if i == angleDestinationStart {
			inAngleDestination = true
			continue
		}
		if current == '\\' && i+1 < len(source) {
			i++
			continue
		}
		if inAngleDestination {
			if current == '>' {
				inAngleDestination = false
			}
			continue
		}
		if quote != 0 {
			if current == quote {
				quote = 0
			}
			continue
		}
		if ignoreQuoted && i > start && isHTMLSpace(source[i-1]) &&
			(current == '"' || current == '\'') {
			quote = current
			continue
		}

		switch current {
		case opening:
			depth++
		case closing:
			depth--
			if depth == 0 {
				return i + 1, true
			}
		}
	}

	return len(source), false
}

// isHTMLSpaceはvalueがMarkdownとHTMLで使うASCII空白文字のいずれかかを返す。
func isHTMLSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\n', '\f', '\r':
		return true
	default:
		return false
	}
}

// rawHTMLNodeRangeはraw HTMLノードがソース中で占める範囲を返す。
func rawHTMLNodeRange(node ast.Node) (byteRange, bool) {
	switch typedNode := node.(type) {
	case *ast.HTMLBlock:
		lines := typedNode.Lines()
		if lines.Len() == 0 {
			return byteRange{}, false
		}

		stop := lines.At(lines.Len() - 1).Stop
		if typedNode.HasClosure() {
			stop = typedNode.ClosureLine.Stop
		}

		return byteRange{start: lines.At(0).Start, stop: stop}, true
	case *ast.RawHTML:
		if typedNode.Segments.Len() == 0 {
			return byteRange{}, false
		}

		first := typedNode.Segments.At(0)
		last := typedNode.Segments.At(typedNode.Segments.Len() - 1)

		return byteRange{start: first.Start, stop: last.Stop}, true
	default:
		return byteRange{}, false
	}
}

// rawHTMLNodeRangesは、解析済み文書がraw HTMLとして識別したすべてのソース範囲を返す。
func rawHTMLNodeRanges(document ast.Node) []byteRange {
	var ranges []byteRange

	// 下のウォーカーは失敗しないため、ast.Walkが返すエラーはnilにしかならない。
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && node.Kind() == ast.KindImage {
			// 画像の子はaltテキストに使われ、HTML要素としては描画されない。
			return ast.WalkSkipChildren, nil
		}
		if entering && (node.Kind() == ast.KindHTMLBlock || node.Kind() == ast.KindRawHTML) {
			if sourceRange, ok := rawHTMLNodeRange(node); ok {
				ranges = append(ranges, sourceRange)
			}
		}

		return ast.WalkContinue, nil
	})

	return ranges
}

// rawHTMLScanOptionsはscanRawHTMLRangesが本文のraw HTMLから読み取るものを選ぶ。
type rawHTMLScanOptions struct {
	// rawHTMLRangesはMarkdownパーサーがraw HTMLとして認識したソース。エスケープされた
	// テキストやMarkdownのコードとして見せているタグはこの外にあり、ここでは要素を指さない。
	rawHTMLRanges []byteRange

	// protectElementは、囲まれたソースを呼び出し元が集めるraw text要素を選ぶ。
	protectElement func(tokenType html.TokenType, name string, raw []byte) bool

	// discardElementは、サニタイズポリシーが中身ごと落とす要素を選ぶ。bluemondayの
	// カウンターに従い、選択された終了タグは名前が異なっても開始の深さを1つ閉じる。
	discardElement func(name string) bool

	// acceptTokenは、そのソース範囲自体を呼び出し元が集めるHTMLトークンを選ぶ。
	acceptToken func(tokenType html.TokenType, name string) bool
}

// tokenizerRawTextElementsは、HTMLトークナイザーが中身をマークアップではなくraw textと
// して読む要素のセット。その中に書かれたタグは、対応する終了タグまでテキストになる。
// golang.org/x/net/htmlがraw text・RCDATAへ切り替える対象と同じものを持つ。
var tokenizerRawTextElements = map[string]bool{
	"iframe": true, "noembed": true, "noframes": true, "noscript": true, "plaintext": true,
	"script": true, "style": true, "textarea": true, "title": true, "xmp": true,
}

// rawHTMLScannerは1つの本文のraw HTMLをたどり、呼び出し元が求めた範囲を集める。
type rawHTMLScanner struct {
	source        []byte
	options       rawHTMLScanOptions
	rawTextSource []byte

	// recognizedは走査を限定するraw HTMLの範囲で、正規化済みのもの
	recognized []byteRange

	// recognizedCursorは、raw textをたどる間に見つけたトークンが認識済みのraw HTMLの中に
	// あるかを判定する。位置はソース順に届く
	recognizedCursor byteRangeCursor

	elementRanges []byteRange
	tokenRanges   []byteRange

	// openProtectedは、まだ開いている保護要素のelementRanges上のインデックスを、要素名
	// ごとに開いた順で持つ
	openProtected map[string][]int

	discardedRangeStart   int
	discardedElementCount int
	discardingContent     bool

	position int
}

// scanRawHTMLRangesは本文のraw HTMLを読み、2種類の範囲を返す。elementRangesは
// protectElementまたはdiscardElementが選んだ要素が囲む範囲、tokenRangesはacceptTokenが
// 選んだHTMLトークンの範囲である。
//
// トークナイズするのはMarkdownパーサーがraw HTMLとして認識したソースだけで、認識済みの範囲を
// 1つずつ読む。その間にあるのはMarkdownのテキストで、画面にはそこに要素が出ないうえ、読むと
// 壊れたタグがその後ろに書かれた認識済みのタグを飲み込んでしまう。読み飛ばすことで、壊れたタグを
// 含む本文でもトークナイズの処理量が本文の長さに比例する。
//
// 範囲の外へ届くのはraw textの要素だけである。トークナイザーはその開始タグの後ろを中身として
// 読むため、走査は対応する終了タグまでその中身をたどってから認識済みの範囲へ戻る。
//
// 2つを同じ走査から得るのは、呼び出し元が本文について両方を一度に必要とするためであり、また
// raw textの要素がその後ろの読み方を変えるので、別々に集めると同じ走査を2回行うことになる
// ためである。
func scanRawHTMLRanges(source []byte, options rawHTMLScanOptions) (elementRanges []byteRange, tokenRanges []byteRange) {
	recognized := normalizeByteRanges(options.rawHTMLRanges)
	if len(recognized) == 0 {
		return nil, nil
	}

	scanner := &rawHTMLScanner{
		source:              source,
		options:             options,
		recognized:          recognized,
		recognizedCursor:    byteRangeCursor{ranges: recognized},
		openProtected:       map[string][]int{},
		discardedRangeStart: -1,
	}
	scanner.scan()

	return scanner.elementRanges, scanner.tokenRanges
}

// scanは認識済みの範囲を順に読み、本文が開いたままにした状態を最後に片付ける。
func (s *rawHTMLScanner) scan() {
	for _, recognized := range s.recognized {
		if recognized.stop <= s.position {
			continue
		}
		if s.position < recognized.start {
			s.position = recognized.start
		}
		for s.position < recognized.stop {
			if !s.readRange(recognized.stop) {
				s.position = recognized.stop
			}
		}
	}
	if s.discardingContent {
		s.elementRanges = append(
			s.elementRanges,
			byteRange{start: s.discardedRangeStart, stop: len(s.source)},
		)
	}
}

// readRangeは現在位置からstopまでのソースをトークナイズする。raw text要素の中身をたどる
// ために途中で止まったかを返す。その場合、位置はstopではなくその中身の後ろにある。
func (s *rawHTMLScanner) readRange(stop int) bool {
	tokenizer := html.NewTokenizer(bytes.NewReader(s.source[s.position:stop]))

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			return false
		}

		tokenStart := s.position
		s.position += len(tokenizer.Raw())

		name := tagName(tokenizer, tokenType)
		s.handleToken(tokenType, name, tokenizer.Raw(), tokenStart, s.position)

		if tokenType == html.StartTagToken && tokenizerRawTextElements[name] {
			s.followRawText(name, tokenStart)

			return true
		}
	}
}

// followRawTextはraw textを、対応する終了タグかソースの末尾まで読み進める。
// トークナイザーにはHTML外の開き山括弧を伏せた同じ長さのコピーを渡す。
// Markdownがエスケープしたテキストとして描画するタグはraw textを終了させず、scriptの
// トークナイザーのエスケープ状態も変えない。コピーは一度だけ作成して前方へ読み進めるため、
// エスケープされた終了タグが多数あっても再走査は増えない。
func (s *rawHTMLScanner) followRawText(name string, tagStart int) {
	if s.rawTextSource == nil {
		s.rawTextSource = bytes.Clone(s.source)
		for _, span := range unprotectedSpans(len(s.source), s.recognized) {
			for position := span.start; position < span.stop; position++ {
				if s.rawTextSource[position] == '<' {
					s.rawTextSource[position] = ' '
				}
			}
		}
	}
	tokenizer := html.NewTokenizer(bytes.NewReader(s.rawTextSource[tagStart:]))
	position := tagStart

	// ここでは開始タグを2度目に読む。呼び出し元が処理済みである。
	if tokenizer.Next() == html.ErrorToken {
		s.position = len(s.source)

		return
	}
	position += len(tokenizer.Raw())

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			s.position = len(s.source)

			return
		}

		tokenStart := position
		position += len(tokenizer.Raw())
		if tokenType != html.EndTagToken || tagName(tokenizer, tokenType) != name {
			continue
		}

		s.position = position
		if s.recognizedCursor.contains(tokenStart, position) {
			s.handleToken(tokenType, name, tokenizer.Raw(), tokenStart, position)
			return
		}

	}
}

// tagNameはタグのトークンの名前を小文字で返す。それ以外では空文字列を返す。
func tagName(tokenizer *html.Tokenizer, tokenType html.TokenType) string {
	switch tokenType {
	case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
		nameBytes, _ := tokenizer.TagName()

		return string(nameBytes)
	default:
		return ""
	}
}

// handleTokenは、認識済みのraw HTMLのトークン1つが何をもたらすかを記録する。
func (s *rawHTMLScanner) handleToken(
	tokenType html.TokenType,
	name string,
	raw []byte,
	start int,
	stop int,
) {
	if s.options.acceptToken(tokenType, name) {
		s.tokenRanges = append(s.tokenRanges, byteRange{start: start, stop: stop})
	}

	isTag := tokenType == html.StartTagToken ||
		tokenType == html.EndTagToken ||
		tokenType == html.SelfClosingTagToken
	if !isTag {
		return
	}

	if s.options.discardElement(name) {
		s.handleDiscardedTag(tokenType, start, stop)
	}

	if s.discardingContent || !s.options.protectElement(tokenType, name, raw) {
		return
	}

	switch tokenType {
	case html.StartTagToken:
		s.openProtected[name] = append(s.openProtected[name], len(s.elementRanges))
		s.elementRanges = append(s.elementRanges, byteRange{start: start, stop: len(s.source)})
	case html.SelfClosingTagToken:
		s.elementRanges = append(s.elementRanges, byteRange{start: start, stop: stop})
	case html.EndTagToken:
		indices := s.openProtected[name]
		if len(indices) > 0 {
			s.elementRanges[indices[len(indices)-1]].stop = stop
			s.openProtected[name] = indices[:len(indices)-1]
		}
	}
}

// handleDiscardedTagは、中身ごと落とす要素についてbluemondayのカウンターに従う。選択
// された終了タグは、選択された開始の深さを1つ閉じる。
func (s *rawHTMLScanner) handleDiscardedTag(tokenType html.TokenType, start int, stop int) {
	switch tokenType {
	case html.StartTagToken:
		if !s.discardingContent {
			s.discardedRangeStart = start
		}
		s.discardingContent = true
		s.discardedElementCount++
	case html.SelfClosingTagToken:
		s.elementRanges = append(s.elementRanges, byteRange{start: start, stop: stop})
	case html.EndTagToken:
		s.discardedElementCount--
		if s.discardingContent && s.discardedElementCount == 0 {
			s.elementRanges = append(
				s.elementRanges,
				byteRange{start: s.discardedRangeStart, stop: stop},
			)
			s.discardedRangeStart = -1
			s.discardingContent = false
		}
	}
}
