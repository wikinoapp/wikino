package markup

import (
	"crypto/rand"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
)

// visibleAttachmentRefs distinguishes use sites sharing an attachment ID by appending unique
// suffixes to their destinations in a temporary rendering. The ordinary rendering already
// settles IDs with just one candidate, so only repeated IDs need this additional pass.
// Parsing is shared, and every changed AST field is restored before returning.
//
// The boolean result is false when no additional pass is needed or rendering fails. The caller
// then keeps the candidates selected by the original rendering and source ranges, preserving
// attachment ownership records when visibility cannot be established.
//
// [Ja] visibleAttachmentRefsは同じ添付IDを持つ参照元を、一時描画のリンク先へ固有の接尾辞を
// 付けて区別する。候補が1つのIDは通常の描画で判定済みなので、追加描画は重複IDがある場合だけ行う。
// 解析結果を共有し、変更したASTのフィールドはすべて戻してから返す。
//
// boolの戻り値は追加描画が不要な場合、または描画に失敗した場合にfalseとなる。
// 呼び出し元は元の描画とソース範囲で選んだ候補を維持し、表示可否を判定できない場合に
// 添付ファイルの参照レコードを失わないようにする。
func visibleAttachmentRefs(source []byte, document ast.Node, refs []attachmentRef) (map[int]bool, bool) {
	counts := make(map[string]int)
	for _, ref := range refs {
		counts[string(ref.match.AttachmentID)]++
	}
	duplicated := false
	for _, count := range counts {
		duplicated = duplicated || count > 1
	}
	if !duplicated {
		return nil, false
	}

	prefix := "Wikino" + rand.Text()
	markers := make(map[int]string)
	visible := make(map[int]bool)
	var htmlRefs []attachmentRef
	for i, ref := range refs {
		if counts[string(ref.match.AttachmentID)] == 1 {
			visible[ref.elementStart] = true
			continue
		}
		markers[ref.elementStart] = prefix + strconv.Itoa(i)
		if ref.match.InHTMLAttribute {
			htmlRefs = append(htmlRefs, ref)
		}
	}

	augmented := source
	var restore []func()
	defer func() {
		for _, undo := range restore {
			undo()
		}
	}()

	// Insert into the segment containing the destination's final byte. Keeping the original
	// bytes and padding preserves multiline raw HTML and Markdown container prefixes.
	//
	// [Ja] リンク先の最後のバイトを含むセグメントへ挿入する。元のバイトとパディングを保持し、
	// 複数行のraw HTMLとMarkdownコンテナーの接頭辞の解釈を変えない。
	markSegment := func(segment gmtext.Segment) gmtext.Segment {
		first := sort.Search(len(htmlRefs), func(i int) bool { return htmlRefs[i].match.Stop > segment.Start })
		if first == len(htmlRefs) || htmlRefs[first].match.Stop > segment.Stop {
			return segment
		}
		start := len(augmented)
		written := segment.Start
		for i := first; i < len(htmlRefs) && htmlRefs[i].match.Stop <= segment.Stop; i++ {
			ref := htmlRefs[i]
			augmented = append(augmented, source[written:ref.match.Stop]...)
			augmented = append(augmented, markers[ref.elementStart]...)
			written = ref.match.Stop
		}
		augmented = append(augmented, source[written:segment.Stop]...)
		return gmtext.NewSegmentPadding(start, len(augmented), segment.Padding)
	}
	markSegments := func(segments *gmtext.Segments) {
		for i := 0; i < segments.Len(); i++ {
			old := segments.At(i)
			marked := markSegment(old)
			if marked != old {
				segments.Set(i, marked)
				restore = append(restore, func() { segments.Set(i, old) })
			}
		}
	}
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Link:
			if marker, ok := markers[n.Pos()]; ok {
				old := n.Destination
				n.Destination = []byte(string(old) + marker)
				restore = append(restore, func() { n.Destination = old })
			}
		case *ast.Image:
			if marker, ok := markers[n.Pos()]; ok {
				old := n.Destination
				n.Destination = []byte(string(old) + marker)
				restore = append(restore, func() { n.Destination = old })
			}
			return ast.WalkSkipChildren, nil
		case *ast.RawHTML:
			markSegments(n.Segments)
		case *ast.HTMLBlock:
			markSegments(n.Lines())
			if n.HasClosure() {
				old := n.ClosureLine
				n.ClosureLine = markSegment(old)
				restore = append(restore, func() { n.ClosureLine = old })
			}
		}
		return ast.WalkContinue, nil
	})
	markedHTML, err := renderSanitized(augmented, document)
	if err != nil {
		slog.Warn("Failed to render attachment reference markers", "error", err)
		return nil, false
	}
	live, known := liveAttachmentIDs(markedHTML, true)
	if !known {
		return nil, false
	}
	for id := range live {
		_, suffix, found := strings.Cut(string(id), prefix)
		if !found {
			continue
		}
		index, err := strconv.Atoi(suffix)
		if err == nil && index >= 0 && index < len(refs) {
			visible[refs[index].elementStart] = true
		}
	}
	return visible, true
}
