package viewmodel

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffLineTypeは差分行の種類を表します
type DiffLineType int

const (
	// DiffLineEqualは変更のない行を表します
	DiffLineEqual DiffLineType = iota
	// DiffLineInsertは追加行を表します
	DiffLineInsert
	// DiffLineDeleteは削除行を表します
	DiffLineDelete
)

// DiffLineは差分表示の1行分のデータです
type DiffLine struct {
	Type      DiffLineType
	Content   string
	OldNumber int
	NewNumber int
}

// DiffBlockは連続する差分行のまとまりです。変更のある箇所とその前後のコンテキスト行を含みます。
type DiffBlock struct {
	Lines []DiffLine
}

// HasChangesはブロック内に変更 (追加・削除) が含まれるかを返します
func (b DiffBlock) HasChanges() bool {
	for _, line := range b.Lines {
		if line.Type == DiffLineInsert || line.Type == DiffLineDelete {
			return true
		}
	}
	return false
}

// bodyDiffContextLinesは本文の差分で変更箇所の前後に表示するコンテキスト行数です。
const bodyDiffContextLines = 3

// DiffSourcesMaxBytesは差分に新旧の本文を埋める上限 (新旧の本文のUTF-8のバイト数の合計) です。
// 本文は記法の色付けのためにブラウザへ送り、ブラウザで解析し直す。上限を超える本文を埋めると、
// 応答が大きくなり、モーダルの表示が受信と解析を待つことになるため、埋めずに色の無い差分にする。
// 上限では、ブラウザでの解析と色付けが100ms未満に収まる。
const DiffSourcesMaxBytes = 100_000

// DiffSourcesは差分の計算に使った、改行を正規化した後の新旧の本文です。差分の各行の行番号は
// この本文の行を指すため、ブラウザはこの本文を解析して各行に記法の色を当てる。
type DiffSources struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// BodyDiffはページ本文の差分です。Sourcesは記法の色付けに使う新旧の本文で、変更が無いときと
// 本文が大きすぎるときはnilになります。
type BodyDiff struct {
	Blocks  []DiffBlock
	Sources *DiffSources
}

// NewBodyDiffはページ本文の新旧の差分を計算します。
func NewBodyDiff(oldText, newText string) BodyDiff {
	oldText = normalizeNewlines(oldText)
	newText = normalizeNewlines(newText)

	diff := BodyDiff{Blocks: computeNormalizedDiffBlocks(oldText, newText, bodyDiffContextLines)}
	if len(diff.Blocks) > 0 && len(oldText)+len(newText) <= DiffSourcesMaxBytes {
		diff.Sources = &DiffSources{Old: oldText, New: newText}
	}

	return diff
}

// ComputeDiffBlocksは2つのテキストの差分を計算し、DiffBlockのスライスとして返します。
// contextLinesは変更箇所の前後に表示するコンテキスト行数です。
func ComputeDiffBlocks(oldText, newText string, contextLines int) []DiffBlock {
	return computeNormalizedDiffBlocks(normalizeNewlines(oldText), normalizeNewlines(newText), contextLines)
}

// computeNormalizedDiffBlocksは改行を正規化した2つのテキストの差分を計算します。
func computeNormalizedDiffBlocks(oldText, newText string, contextLines int) []DiffBlock {
	dmp := diffmatchpatch.New()
	a, b, c := dmp.DiffLinesToChars(oldText, newText)
	diffs := dmp.DiffMain(a, b, false)
	// 整理は行を置き換えた文字の上で行い、編集の位置を行の単位でだけ動かす。差分の断片を行の境目に揃え、
	// 各行の行番号が新旧の本文の行を指すようにするためである。
	diffs = dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCharsToLines(diffs, c)

	lines := buildDiffLines(diffs)
	if len(lines) == 0 {
		return nil
	}

	return groupIntoBlocks(lines, contextLines)
}

// normalizeNewlinesは改行コードを\nに統一し、末尾に改行がなければ追加します。
// ブラウザのフォーム送信では\r\nが使われることがあり、DB内の既存データと改行コードが異なると
// 同じ行内容でも差分として検出されるためです。
func normalizeNewlines(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text
}

// buildDiffLinesはdiffの結果からDiffLineのスライスを構築します
func buildDiffLines(diffs []diffmatchpatch.Diff) []DiffLine {
	var lines []DiffLine
	oldNum := 1
	newNum := 1

	for _, d := range diffs {
		if d.Text == "" {
			continue
		}

		// 末尾の改行を除去して行に分割する。改行1つだけの断片も空行1行なので、行番号を進める。
		text := strings.TrimSuffix(d.Text, "\n")
		splitLines := strings.Split(text, "\n")

		switch d.Type {
		case diffmatchpatch.DiffEqual:
			for _, l := range splitLines {
				lines = append(lines, DiffLine{
					Type:      DiffLineEqual,
					Content:   l,
					OldNumber: oldNum,
					NewNumber: newNum,
				})
				oldNum++
				newNum++
			}
		case diffmatchpatch.DiffDelete:
			for _, l := range splitLines {
				lines = append(lines, DiffLine{
					Type:      DiffLineDelete,
					Content:   l,
					OldNumber: oldNum,
				})
				oldNum++
			}
		case diffmatchpatch.DiffInsert:
			for _, l := range splitLines {
				lines = append(lines, DiffLine{
					Type:      DiffLineInsert,
					Content:   l,
					NewNumber: newNum,
				})
				newNum++
			}
		}
	}

	return lines
}

// groupIntoBlocksは差分行をコンテキスト行数に基づいてブロックに分割します
func groupIntoBlocks(lines []DiffLine, contextLines int) []DiffBlock {
	// 変更行のインデックスを収集
	changeIndices := make([]int, 0)
	for i, l := range lines {
		if l.Type != DiffLineEqual {
			changeIndices = append(changeIndices, i)
		}
	}

	if len(changeIndices) == 0 {
		return nil
	}

	// 変更行を含む範囲を計算 (コンテキスト行を含む)
	type span struct {
		start, end int
	}
	spans := make([]span, 0)

	for _, idx := range changeIndices {
		start := idx - contextLines
		if start < 0 {
			start = 0
		}
		end := idx + contextLines + 1
		if end > len(lines) {
			end = len(lines)
		}

		// 前のスパンとマージ可能ならマージ
		if len(spans) > 0 && start <= spans[len(spans)-1].end {
			spans[len(spans)-1].end = end
		} else {
			spans = append(spans, span{start: start, end: end})
		}
	}

	blocks := make([]DiffBlock, len(spans))
	for i, s := range spans {
		blocks[i] = DiffBlock{
			Lines: lines[s.start:s.end],
		}
	}

	return blocks
}
