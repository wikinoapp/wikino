package viewmodel

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffLineType は差分行の種類を表します
type DiffLineType int

const (
	// DiffLineEqual は変更のない行を表します
	DiffLineEqual DiffLineType = iota
	// DiffLineInsert は追加行を表します
	DiffLineInsert
	// DiffLineDelete は削除行を表します
	DiffLineDelete
)

// DiffLine は差分表示の1行分のデータです
type DiffLine struct {
	Type      DiffLineType
	Content   string
	OldNumber int
	NewNumber int
}

// DiffBlock は連続する差分行のまとまりです。変更のある箇所とその前後のコンテキスト行を含みます。
type DiffBlock struct {
	Lines []DiffLine
}

// HasChanges はブロック内に変更（追加・削除）が含まれるかを返します
func (b DiffBlock) HasChanges() bool {
	for _, line := range b.Lines {
		if line.Type == DiffLineInsert || line.Type == DiffLineDelete {
			return true
		}
	}
	return false
}

// ComputeDiffBlocks は2つのテキストの差分を計算し、DiffBlockのスライスとして返します。
// contextLines は変更箇所の前後に表示するコンテキスト行数です。
func ComputeDiffBlocks(oldText, newText string, contextLines int) []DiffBlock {
	// 改行コードを正規化する。ブラウザのフォーム送信では\r\nが使われることがあり、
	// DB内の既存データと改行コードが異なると同じ行内容でも差分として検出される。
	oldText = normalizeNewlines(oldText)
	newText = normalizeNewlines(newText)

	dmp := diffmatchpatch.New()
	a, b, c := dmp.DiffLinesToChars(oldText, newText)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, c)
	diffs = dmp.DiffCleanupSemantic(diffs)

	lines := buildDiffLines(diffs)
	if len(lines) == 0 {
		return nil
	}

	return groupIntoBlocks(lines, contextLines)
}

// normalizeNewlines は改行コードを\nに統一し、末尾に改行がなければ追加します
func normalizeNewlines(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text
}

// buildDiffLines はdiffの結果からDiffLineのスライスを構築します
func buildDiffLines(diffs []diffmatchpatch.Diff) []DiffLine {
	var lines []DiffLine
	oldNum := 1
	newNum := 1

	for _, d := range diffs {
		// 末尾の改行を除去して行に分割
		text := strings.TrimSuffix(d.Text, "\n")
		if text == "" {
			continue
		}

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

// groupIntoBlocks は差分行をコンテキスト行数に基づいてブロックに分割します
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

	// 変更行を含む範囲を計算（コンテキスト行を含む）
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
