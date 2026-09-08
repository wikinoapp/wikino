package markup

import (
	"bytes"
	"cmp"
	"slices"
	"sort"

	"github.com/yuin/goldmark/ast"
	"golang.org/x/net/html"
)

// byteRange is a half-open range of byte offsets into a Markdown body.
//
// [Ja] byteRange は Markdown 本文へのバイト位置の半開区間。
type byteRange struct {
	start int
	stop  int
}

// normalizeByteRanges sorts ranges by their start offset and merges ranges that overlap or touch.
// Callers that compare positions in source order can then advance one cursor through the result.
//
// [Ja] normalizeByteRanges は範囲を開始位置順に並べ、重なる範囲と接する範囲を統合する。
// ソース順に位置を照合する呼び出し元は、その結果を 1 つのカーソルで前へ進められる。
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

// byteRangeCursor compares source ranges that arrive in monotonically increasing order with a
// normalized range set. Each protected range is passed at most once across all comparisons.
//
// [Ja] byteRangeCursor はソース順に単調増加して届く範囲を、正規化済みの範囲集合と照合する。
// 一連の照合全体で、それぞれの保護範囲を通過するのは高々 1 回である。
type byteRangeCursor struct {
	ranges []byteRange
	index  int
}

// overlaps reports whether [start, stop) overlaps the cursor's current or later range.
//
// [Ja] overlaps は [start, stop) がカーソルの現在位置以降の範囲と重なるかを返す。
func (cursor *byteRangeCursor) overlaps(start int, stop int) bool {
	for cursor.index < len(cursor.ranges) && cursor.ranges[cursor.index].stop <= start {
		cursor.index++
	}

	return cursor.index < len(cursor.ranges) &&
		start < cursor.ranges[cursor.index].stop &&
		cursor.ranges[cursor.index].start < stop
}

// contains reports whether one range of the cursor holds the whole of [start, stop). A normalized
// range set has no two ranges that overlap or touch, so a span reaching past one of them is held by
// none of them.
//
// [Ja] contains は [start, stop) の全体をカーソルのいずれかの範囲が含むかを返す。正規化済みの
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

// unprotectedSpans returns the parts of [0, length) that ranges leaves out, in source order.
// ranges has to be normalized.
//
// A caller looking for a pattern that has to sit inside one of these parts applies it to each span
// rather than to the whole source. That is what rendering does with the text of a body: protected
// syntax splits it into separate runs, and each run is read on its own. Matching the whole source
// instead would let a match that starts before a protected range and ends after it cover, and so
// hide, a match that the screen finds inside one of the runs.
//
// [Ja] unprotectedSpans は [0, length) のうち ranges が覆っていない部分を、ソース順に返す。
// ranges は正規化済みである必要がある。
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

// byteRangesHold reports whether one of ranges holds position. ranges has to be normalized, which
// lets a caller comparing positions that do not arrive in source order search them instead of
// walking a cursor forward.
//
// [Ja] byteRangesHold は、ranges のいずれかが position を含むかを返す。ranges は正規化済みである
// 必要があり、これにより照合する位置がソース順に届かない呼び出し元も、カーソルを前へ進めるのでは
// なく探索できる。
func byteRangesHold(ranges []byteRange, position int) bool {
	index := sort.Search(len(ranges), func(i int) bool { return ranges[i].stop > position })

	return index < len(ranges) && ranges[index].start <= position
}

// scanMarkdownCodeRanges returns the ranges of source that Markdown syntax marks as code: the info
// string and body lines of a fenced code block or the lines of an indented code block, and the text
// of a code span. document is the parse the caller already holds, produced by the parser rendering
// uses, so a body reads the same way here as it does on the screen.
//
// Raw HTML code and pre elements are handled by scanRenderedWikilinkRanges, where the actual HTML
// tree determines which text they enclose.
//
// [Ja] scanMarkdownCodeRanges は source のうち Markdown の構文がコードとしている範囲を返す。
// フェンス付きコードブロックの情報文字列と本文各行または字下げコードブロックの各行、そして
// コードスパンの中身である。document は呼び出し元が既に持っている解析結果で、レンダリングが使う
// パーサーが作ったものなので、本文の読み方は画面上と一致する。
//
// raw HTML の code 要素・pre 要素は scanRenderedWikilinkRanges が扱い、実際の HTML ツリーから
// それらが囲むテキストを判定する。
func scanMarkdownCodeRanges(document ast.Node) []byteRange {
	var ranges []byteRange
	// The walker below never fails, so the error ast.Walk returns can only be nil.
	//
	// [Ja] 下のウォーカーは失敗しないため、ast.Walk が返すエラーは nil にしかならない。
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

// markdownLink is what this package reads from an ast.Link or an ast.Image. Goldmark gives the two
// nodes the same fields but no common type to reach them through.
//
// [Ja] markdownLink は本パッケージが ast.Link・ast.Image から読む部分。goldmark は 2 つのノードに
// 同じフィールドを持たせているが、それらへ到達するための共通の型は用意していない。
type markdownLink struct {
	// reference is the reference the link names, and nil when the link carries its own inline
	// destination.
	//
	// [Ja] reference はリンクが指す参照で、リンクが自身のインラインのリンク先を持つ場合は nil
	reference *ast.ReferenceLink

	// destination is the destination the parser read.
	//
	// [Ja] destination はパーサーが読んだリンク先
	destination []byte
}

// markdownLinkOf returns what node holds if node is a Markdown link or image.
//
// [Ja] markdownLinkOf は node が Markdown のリンク・画像であれば、その持ち物を返す。
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

// markdownLinkSourceRanges keeps the whole syntax, the label it encloses, and the replaceable
// destination separately. A reference link has no destination in its own source range.
//
// The label is kept apart from the rest because rendering can drop the element while keeping what
// the label holds, which then reaches the reader as ordinary text.
//
// [Ja] markdownLinkSourceRanges は構文全体、それが囲むラベル、置換可能なリンク先の範囲を分けて
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

// scanMarkdownLinkRanges computes children before their enclosing links. Raw HTML, code spans,
// and complete nested links/images are opaque to the enclosing label's bracket scan. Each child's
// range replaces its collected descendants, so unrelated ranges are never copied into a label.
//
// [Ja] scanMarkdownLinkRanges は外側のリンクより先に子の範囲を計算する。raw HTML、コードスパン、
// 入れ子のリンク・画像全体は、外側ラベルの角括弧の走査では読み飛ばす。各子の範囲で収集済みの
// 子孫を置き換えるため、無関係な範囲をラベルごとにコピーすることはない。
func scanMarkdownLinkRanges(source []byte, document ast.Node) map[ast.Node]markdownLinkSourceRanges {
	ranges := make(map[ast.Node]markdownLinkSourceRanges)
	var labelStarts []int
	var syntaxRanges []byteRange

	// The walker never fails; leaving a link happens after all its children have been visited.
	//
	// [Ja] ウォーカーは失敗しない。リンクから退出する時点では、子をすべて走査済みである。
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

// markdownLinkNodeRange returns the source range of a Markdown link or image node. Goldmark keeps
// only its start offset, so this follows the balanced label and then whatever the shape of the
// link says comes after it: an inline destination in parentheses, the label of a full or collapsed
// reference, or nothing at all.
//
// A shortcut reference ends at its label. Reading a following "[...]" as part of it would swallow
// text the parser left outside the link, such as the wiki link of "[a][[page]]".
//
// labelSyntaxRanges holds parsed syntax whose brackets do not delimit the label. Only the label
// needs it: an inline destination and the label of a reference are read raw.
//
// [Ja] markdownLinkNodeRange は Markdown のリンク・画像ノードがソース中で占める範囲を返す。
// goldmark が保持するのは開始位置だけなので、対応の取れたラベルをたどり、その後ろはリンクの形が
// 示すものをたどる。丸括弧に囲まれたインラインのリンク先か、full・collapsed の参照のラベルか、
// あるいは何も続かないかである。
//
// ショートカット参照はラベルで終わる。続く "[...]" をその一部として読むと、"[a][[ページ]]" の
// Wiki リンクのような、パーサーがリンクの外に残したテキストまで飲み込んでしまう。
//
// labelSyntaxRanges はラベルを区切らない角括弧を持つ解析済み構文の範囲である。必要とするのは
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
			// Reuse the parsed destination's range so parentheses inside it cannot delimit the
			// link, including when quote/list prefixes separate it from the opening parenthesis.
			//
			// [Ja] 解析済みリンク先の範囲を使い、その中の丸括弧でリンクを区切らないようにする。
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

// markdownLinkDestinationRange returns the source range the destination of a Markdown link or
// image occupies, leaving out the surrounding syntax and any title that follows. A reference link
// yields nothing: its destination is written in a link reference definition rather than in the
// link itself, and linkReferenceDefinitionDestinationRange returns it from there.
//
// [Ja] markdownLinkDestinationRange は Markdown のリンク・画像のリンク先がソース中で占める範囲を
// 返す。周囲の構文と、後ろに続くタイトルは含めない。参照リンクでは何も返さない。そのリンク先は
// リンク自身ではなくリンク参照定義に書かれており、linkReferenceDefinitionDestinationRange が
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

// linkReferenceDefinitionDestinationRange returns the source range the destination of a link
// reference definition occupies, leaving out the surrounding syntax and any title that follows.
//
// This is where the destination of a reference link is written, so it is the one place a caller
// rewriting where a body points can reach for that shape of link. The link itself carries no
// destination of its own to replace, and several links can name the same definition.
//
// [Ja] linkReferenceDefinitionDestinationRange は、リンク参照定義のリンク先がソース中で占める範囲
// を返す。周囲の構文と、後ろに続くタイトルは含めない。
//
// 参照リンクのリンク先が書かれているのはここであり、その形のリンクについて本文の指す先を書き換える
// 呼び出し元が触れられる唯一の場所である。リンク自身は置き換えるリンク先を持たず、1 つの定義を
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

	// The label of a link reference definition is read raw, so no code span resolves inside it and
	// a "]" there closes the label whatever surrounds it.
	//
	// [Ja] リンク参照定義のラベルは生のまま読まれるため、その中でコードスパンは解決されず、
	// そこにある "]" は何に囲まれていてもラベルを閉じる。
	labelStop, ok := scanBalancedDelimiter(source, labelStart, '[', ']', false, nil)
	if !ok || labelStop >= len(source) || source[labelStop] != ':' {
		return byteRange{}, false
	}

	return destinationRangeAt(source, node, labelStop+1, definition.Destination)
}

// destinationRangeAt returns the source range of the destination that starts at the first byte
// after start which is not a space, written either bare or between angle brackets. The block's
// line segments exclude quote/list prefixes, just as the parser's input does. Destinations fit
// in one segment; only the whitespace before them can cross lines.
//
// The range is returned only when it covers destination, the bytes the parser read. Anything else
// means the offsets found here name something other than the destination, and a caller replacing
// them would write over the wrong bytes.
//
// [Ja] destinationRangeAt は、start 以降で最初に空白でないバイトから始まるリンク先がソース中で
// 占める範囲を返す。リンク先は裸で書かれていても山括弧で囲まれていてもよい。ブロックの行セグメント
// はパーサーへの入力と同じく引用・リストの接頭辞を除いている。リンク先は 1 セグメント内に収まり、
// その前の空白だけが行をまたぐ。
//
// 範囲を返すのは、それがパーサーの読んだバイト列である destination を覆っている場合だけである。
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

// markdownLinkLabelRange returns the start of a Markdown link or image node and the offset just
// past the "]" that closes its label. labelSyntaxRanges holds parsed syntax inside this label,
// whose brackets neither open nor close the label.
//
// [Ja] markdownLinkLabelRange は Markdown のリンク・画像ノードの開始位置と、そのラベルを閉じる
// "]" の直後の位置を返す。labelSyntaxRanges はラベル内の解析済み構文で、その中の角括弧では
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

// markdownLinkLabelContentRange returns the range between the brackets of a Markdown link or image
// label, leaving the brackets themselves out. Sanitization can drop the element a link renders to
// while keeping this content, which then reaches the reader as ordinary text.
//
// [Ja] markdownLinkLabelContentRange は、Markdown のリンク・画像のラベルの角括弧に挟まれた範囲を
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

// scanBalancedDelimiter returns the offset after the closing delimiter paired with source[start].
// Backslash escapes are skipped, quoted text can optionally hide delimiters, and a byte inside one
// of skipRanges is never a delimiter. skipRanges has to be normalized and holds parsed syntax or
// a destination whose contents are opaque to this scan.
//
// [Ja] scanBalancedDelimiter は source[start] と対になる閉じ区切りの直後の位置を返す。
// バックスラッシュによるエスケープを飛ばし、必要に応じて引用符内の区切りも無視する。skipRanges の
// いずれかの中にあるバイトは区切りにならない。skipRanges は正規化済みである必要があり、この走査で
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

	// Parentheses inside an angle-bracket destination are ordinary destination bytes, not
	// delimiters of the surrounding inline link. Only the first non-space byte can open that
	// destination shape.
	//
	// [Ja] 山括弧付きリンク先の中の丸括弧は、外側のインラインリンクの区切りではなくリンク先の
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

	// Enter the skipped ranges at the first one that can still hold start, so that a body with
	// many of them does not cost every scan a walk from the beginning of the set.
	//
	// [Ja] 飛ばす範囲には、start を含みうる最初のものから入る。そうした範囲の多い本文で、走査の
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

// isHTMLSpace reports whether value is one of the ASCII whitespace bytes Markdown and HTML use.
//
// [Ja] isHTMLSpace は value が Markdown と HTML で使う ASCII 空白文字のいずれかかを返す。
func isHTMLSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\n', '\f', '\r':
		return true
	default:
		return false
	}
}

// rawHTMLNodeRange returns the source range occupied by a raw HTML node.
//
// [Ja] rawHTMLNodeRange は raw HTML ノードがソース中で占める範囲を返す。
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

// rawHTMLNodeRanges returns every source range the parsed document identifies as raw HTML.
//
// [Ja] rawHTMLNodeRanges は、解析済み文書が raw HTML として識別したすべてのソース範囲を返す。
func rawHTMLNodeRanges(document ast.Node) []byteRange {
	var ranges []byteRange

	// The walker below never fails, so the error ast.Walk returns can only be nil.
	//
	// [Ja] 下のウォーカーは失敗しないため、ast.Walk が返すエラーは nil にしかならない。
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && node.Kind() == ast.KindImage {
			// Image children contribute alt text, never rendered HTML elements.
			//
			// [Ja] 画像の子は alt テキストに使われ、HTML 要素としては描画されない。
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

// rawHTMLScanOptions selects what scanRawHTMLRanges reads out of the raw HTML of a body.
//
// [Ja] rawHTMLScanOptions は scanRawHTMLRanges が本文の raw HTML から読み取るものを選ぶ。
type rawHTMLScanOptions struct {
	// rawHTMLRanges holds the source the Markdown parser recognized as raw HTML. A tag the body
	// shows as escaped text or as Markdown code sits outside it and names no element here.
	//
	// [Ja] rawHTMLRanges は Markdown パーサーが raw HTML として認識したソース。エスケープされた
	// テキストや Markdown のコードとして見せているタグはこの外にあり、ここでは要素を指さない。
	rawHTMLRanges []byteRange

	// protectElement selects raw-text elements whose enclosed source the caller collects.
	//
	// [Ja] protectElement は、囲まれたソースを呼び出し元が集める raw text 要素を選ぶ。
	protectElement func(tokenType html.TokenType, name string, raw []byte) bool

	// discardElement selects the elements the sanitization policy drops together with their
	// content, following bluemonday's counter: any selected end tag closes one selected start
	// level, even when their names differ.
	//
	// [Ja] discardElement は、サニタイズポリシーが中身ごと落とす要素を選ぶ。bluemonday の
	// カウンターに従い、選択された終了タグは名前が異なっても開始の深さを 1 つ閉じる。
	discardElement func(name string) bool

	// acceptToken selects the HTML tokens whose own source range the caller collects.
	//
	// [Ja] acceptToken は、そのソース範囲自体を呼び出し元が集める HTML トークンを選ぶ。
	acceptToken func(tokenType html.TokenType, name string) bool
}

// tokenizerRawTextElements is the set of elements whose content the HTML tokenizer reads as raw
// text rather than as markup, so a tag written inside one is text until the matching end tag.
// It mirrors what golang.org/x/net/html switches into raw text or RCDATA on.
//
// [Ja] tokenizerRawTextElements は、HTML トークナイザーが中身をマークアップではなく raw text と
// して読む要素のセット。その中に書かれたタグは、対応する終了タグまでテキストになる。
// golang.org/x/net/html が raw text・RCDATA へ切り替える対象と同じものを持つ。
var tokenizerRawTextElements = map[string]bool{
	"iframe": true, "noembed": true, "noframes": true, "noscript": true, "plaintext": true,
	"script": true, "style": true, "textarea": true, "title": true, "xmp": true,
}

// rawHTMLScanner walks the raw HTML of one body and collects the ranges its caller asked for.
//
// [Ja] rawHTMLScanner は 1 つの本文の raw HTML をたどり、呼び出し元が求めた範囲を集める。
type rawHTMLScanner struct {
	source        []byte
	options       rawHTMLScanOptions
	rawTextSource []byte

	// recognized holds the raw HTML ranges the scan is limited to, normalized.
	//
	// [Ja] recognized は走査を限定する raw HTML の範囲で、正規化済みのもの
	recognized []byteRange

	// recognizedCursor tells whether a token found while following raw text sits in recognized
	// raw HTML. Positions reach it in source order.
	//
	// [Ja] recognizedCursor は、raw text をたどる間に見つけたトークンが認識済みの raw HTML の中に
	// あるかを判定する。位置はソース順に届く
	recognizedCursor byteRangeCursor

	elementRanges []byteRange
	tokenRanges   []byteRange

	// openProtected holds, per element name, the indices into elementRanges of the protected
	// elements that are still open, in the order they opened.
	//
	// [Ja] openProtected は、まだ開いている保護要素の elementRanges 上のインデックスを、要素名
	// ごとに開いた順で持つ
	openProtected map[string][]int

	discardedRangeStart   int
	discardedElementCount int
	discardingContent     bool

	position int
}

// scanRawHTMLRanges reads the raw HTML of a body and returns two sets of ranges: elementRanges
// holds what the elements selected by protectElement or discardElement enclose, and tokenRanges
// holds the HTML tokens selected by acceptToken.
//
// Only the source the Markdown parser recognized as raw HTML is tokenized, one recognized range at
// a time. What lies between them is Markdown text: the screen shows no element there, and reading
// it would let a malformed tag swallow the recognized tag written after it. Skipping it keeps
// tokenization proportional to the length of the body, including bodies with malformed tags.
//
// A raw-text element is the one thing that reaches past its range: the tokenizer reads what follows
// its start tag as content, so the scan follows that content to the matching end tag before it
// returns to the recognized ranges.
//
// Both sets come from the same walk because a caller wants both of a body at once, and because a
// raw-text element changes how everything after it is read, so collecting them separately would
// cost the same walk twice.
//
// [Ja] scanRawHTMLRanges は本文の raw HTML を読み、2 種類の範囲を返す。elementRanges は
// protectElement または discardElement が選んだ要素が囲む範囲、tokenRanges は acceptToken が
// 選んだ HTML トークンの範囲である。
//
// トークナイズするのは Markdown パーサーが raw HTML として認識したソースだけで、認識済みの範囲を
// 1 つずつ読む。その間にあるのは Markdown のテキストで、画面にはそこに要素が出ないうえ、読むと
// 壊れたタグがその後ろに書かれた認識済みのタグを飲み込んでしまう。読み飛ばすことで、壊れたタグを
// 含む本文でもトークナイズの処理量が本文の長さに比例する。
//
// 範囲の外へ届くのは raw text の要素だけである。トークナイザーはその開始タグの後ろを中身として
// 読むため、走査は対応する終了タグまでその中身をたどってから認識済みの範囲へ戻る。
//
// 2 つを同じ走査から得るのは、呼び出し元が本文について両方を一度に必要とするためであり、また
// raw text の要素がその後ろの読み方を変えるので、別々に集めると同じ走査を 2 回行うことになる
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

// scan reads each recognized range in turn and finishes the state the body left open.
//
// [Ja] scan は認識済みの範囲を順に読み、本文が開いたままにした状態を最後に片付ける。
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

// readRange tokenizes the source from the current position up to stop. It reports whether it
// stopped early to follow the content of a raw-text element, which leaves the position past that
// content rather than at stop.
//
// [Ja] readRange は現在位置から stop までのソースをトークナイズする。raw text 要素の中身をたどる
// ために途中で止まったかを返す。その場合、位置は stop ではなくその中身の後ろにある。
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

// followRawText moves past raw text to its matching end tag or the end of the source.
// The tokenizer sees a same-length copy with non-HTML opening brackets masked. A tag Markdown
// renders as escaped text cannot end raw text or change the script tokenizer's escaped state.
// The copy is built once and consumed forward, even when many escaped closing tags occur.
//
// [Ja] followRawText は raw text を、対応する終了タグかソースの末尾まで読み進める。
// トークナイザーには HTML 外の開き山括弧を伏せた同じ長さのコピーを渡す。
// Markdown がエスケープしたテキストとして描画するタグは raw text を終了させず、script の
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

	// The start tag is read a second time here; the caller has already handled it.
	//
	// [Ja] ここでは開始タグを 2 度目に読む。呼び出し元が処理済みである。
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

// tagName returns the lowercased name of a tag token, and the empty string for anything else.
//
// [Ja] tagName はタグのトークンの名前を小文字で返す。それ以外では空文字列を返す。
func tagName(tokenizer *html.Tokenizer, tokenType html.TokenType) string {
	switch tokenType {
	case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
		nameBytes, _ := tokenizer.TagName()

		return string(nameBytes)
	default:
		return ""
	}
}

// handleToken records what one HTML token of the recognized raw HTML contributes.
//
// [Ja] handleToken は、認識済みの raw HTML のトークン 1 つが何をもたらすかを記録する。
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

// handleDiscardedTag follows bluemonday's counter for the elements it drops together with their
// content, where any selected end tag closes one selected start level.
//
// [Ja] handleDiscardedTag は、中身ごと落とす要素について bluemonday のカウンターに従う。選択
// された終了タグは、選択された開始の深さを 1 つ閉じる。
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
