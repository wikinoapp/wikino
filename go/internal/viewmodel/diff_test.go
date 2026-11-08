package viewmodel_test

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestComputeDiffBlocks(t *testing.T) {
	t.Parallel()

	t.Run("同一テキストの場合は空を返す", func(t *testing.T) {
		t.Parallel()

		text := "line1\nline2\nline3\n"
		blocks := viewmodel.ComputeDiffBlocks(text, text, 3)
		if len(blocks) != 0 {
			t.Errorf("len(blocks) = %d、期待値 = 0", len(blocks))
		}
	})

	t.Run("空テキスト同士の場合は空を返す", func(t *testing.T) {
		t.Parallel()

		blocks := viewmodel.ComputeDiffBlocks("", "", 3)
		if len(blocks) != 0 {
			t.Errorf("len(blocks) = %d、期待値 = 0", len(blocks))
		}
	})

	t.Run("行の追加を検出する", func(t *testing.T) {
		t.Parallel()

		oldText := "line1\nline2\n"
		newText := "line1\nline2\nline3\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("ブロックが1つも無い")
		}

		hasInsert := false
		for _, block := range blocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineInsert && line.Content == "line3" {
					hasInsert = true
				}
			}
		}
		if !hasInsert {
			t.Error("内容が'line3'の追加行が無い")
		}
	})

	t.Run("行の削除を検出する", func(t *testing.T) {
		t.Parallel()

		oldText := "line1\nline2\nline3\n"
		newText := "line1\nline3\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("ブロックが1つも無い")
		}

		hasDelete := false
		for _, block := range blocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineDelete && line.Content == "line2" {
					hasDelete = true
				}
			}
		}
		if !hasDelete {
			t.Error("内容が'line2'の削除行が無い")
		}
	})

	t.Run("行の変更を検出する", func(t *testing.T) {
		t.Parallel()

		oldText := "line1\nold content\nline3\n"
		newText := "line1\nnew content\nline3\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("ブロックが1つも無い")
		}

		hasDelete := false
		hasInsert := false
		for _, block := range blocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineDelete && line.Content == "old content" {
					hasDelete = true
				}
				if line.Type == viewmodel.DiffLineInsert && line.Content == "new content" {
					hasInsert = true
				}
			}
		}
		if !hasDelete {
			t.Error("内容が'old content'の削除行が無い")
		}
		if !hasInsert {
			t.Error("内容が'new content'の追加行が無い")
		}
	})

	t.Run("行番号が正しく設定される", func(t *testing.T) {
		t.Parallel()

		oldText := "line1\nline2\nline3\n"
		newText := "line1\nnew line\nline2\nline3\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("ブロックが1つも無い")
		}

		for _, block := range blocks {
			for _, line := range block.Lines {
				switch line.Type {
				case viewmodel.DiffLineEqual:
					if line.OldNumber == 0 {
						t.Errorf("一致行のOldNumber = %d、期待値 = 0より大きい", line.OldNumber)
					}
					if line.NewNumber == 0 {
						t.Errorf("一致行のNewNumber = %d、期待値 = 0より大きい", line.NewNumber)
					}
				case viewmodel.DiffLineDelete:
					if line.OldNumber == 0 {
						t.Errorf("削除行のOldNumber = %d、期待値 = 0より大きい", line.OldNumber)
					}
				case viewmodel.DiffLineInsert:
					if line.NewNumber == 0 {
						t.Errorf("追加行のNewNumber = %d、期待値 = 0より大きい", line.NewNumber)
					}
				}
			}
		}
	})

	t.Run("コンテキスト行数でブロックが分割される", func(t *testing.T) {
		t.Parallel()

		// 10行のテキストで、行3と行8を変更。contextLines=1なので離れたブロックに分割される
		oldText := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\n"
		newText := "line1\nline2\nchanged3\nline4\nline5\nline6\nline7\nchanged8\nline9\nline10\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 1)

		if len(blocks) != 2 {
			t.Errorf("len(blocks) = %d、期待値 = 2", len(blocks))
		}
	})

	t.Run("近い変更はマージされる", func(t *testing.T) {
		t.Parallel()

		// 行2と行4を変更。contextLines=3なので1つのブロックにマージされる
		oldText := "line1\nline2\nline3\nline4\nline5\n"
		newText := "line1\nchanged2\nline3\nchanged4\nline5\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) != 1 {
			t.Errorf("len(blocks) = %d、期待値 = 1", len(blocks))
		}
	})

	t.Run("空テキストから新規テキストへの差分", func(t *testing.T) {
		t.Parallel()

		blocks := viewmodel.ComputeDiffBlocks("", "new line\n", 3)

		if len(blocks) == 0 {
			t.Fatal("ブロックが1つも無い")
		}

		hasInsert := false
		for _, block := range blocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineInsert {
					hasInsert = true
				}
			}
		}
		if !hasInsert {
			t.Error("追加行が無い")
		}
	})

	t.Run("末尾改行なしのテキストに行を追加した場合に余分な空行が含まれない", func(t *testing.T) {
		t.Parallel()

		oldText := "foo"
		newText := "foo\nbuz"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("ブロックが1つも無い")
		}

		for _, block := range blocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineInsert && line.Content == "" {
					t.Error("予期しない空の追加行")
				}
			}
		}

		// "foo"がEqual、"buz"がInsertであることを確認
		hasEqual := false
		hasInsert := false
		for _, block := range blocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineEqual && line.Content == "foo" {
					hasEqual = true
				}
				if line.Type == viewmodel.DiffLineInsert && line.Content == "buz" {
					hasInsert = true
				}
			}
		}
		if !hasEqual {
			t.Error("内容が'foo'の一致行が無い")
		}
		if !hasInsert {
			t.Error("内容が'buz'の追加行が無い")
		}
	})

	t.Run("CRLFとLFの混在でも同じ行内容はEqualとして検出される", func(t *testing.T) {
		t.Parallel()

		oldText := "foo\r\nbar\r\n"
		newText := "foo\nbar\nbuz\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("ブロックが1つも無い")
		}

		var lines []viewmodel.DiffLine
		for _, block := range blocks {
			lines = append(lines, block.Lines...)
		}

		// "foo"と"bar"はEqualであること
		equalFoo := false
		equalBar := false
		insertBuz := false
		for _, line := range lines {
			if line.Type == viewmodel.DiffLineEqual && line.Content == "foo" {
				equalFoo = true
			}
			if line.Type == viewmodel.DiffLineEqual && line.Content == "bar" {
				equalBar = true
			}
			if line.Type == viewmodel.DiffLineInsert && line.Content == "buz" {
				insertBuz = true
			}
			// "foo"や"bar"がDelete/Insertとして検出されていないこと
			if (line.Type == viewmodel.DiffLineDelete || line.Type == viewmodel.DiffLineInsert) &&
				(line.Content == "foo" || line.Content == "bar") {
				t.Errorf("'%s'が一致していない: %v", line.Content, line.Type)
			}
		}
		if !equalFoo {
			t.Error("'foo'が一致行になっていない")
		}
		if !equalBar {
			t.Error("'bar'が一致行になっていない")
		}
		if !insertBuz {
			t.Error("'buz'が追加行になっていない")
		}
	})

	t.Run("テキストから空テキストへの差分", func(t *testing.T) {
		t.Parallel()

		blocks := viewmodel.ComputeDiffBlocks("old line\n", "", 3)

		if len(blocks) == 0 {
			t.Fatal("ブロックが1つも無い")
		}

		hasDelete := false
		for _, block := range blocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineDelete {
					hasDelete = true
				}
			}
		}
		if !hasDelete {
			t.Error("削除行が無い")
		}
	})
}

func TestDiffBlock_HasChanges(t *testing.T) {
	t.Parallel()

	t.Run("変更がある場合はtrueを返す", func(t *testing.T) {
		t.Parallel()

		block := viewmodel.DiffBlock{
			Lines: []viewmodel.DiffLine{
				{Type: viewmodel.DiffLineEqual, Content: "line1"},
				{Type: viewmodel.DiffLineInsert, Content: "new line"},
			},
		}
		if !block.HasChanges() {
			t.Error("HasChanges() = false、期待値 = true")
		}
	})

	t.Run("変更がない場合はfalseを返す", func(t *testing.T) {
		t.Parallel()

		block := viewmodel.DiffBlock{
			Lines: []viewmodel.DiffLine{
				{Type: viewmodel.DiffLineEqual, Content: "line1"},
				{Type: viewmodel.DiffLineEqual, Content: "line2"},
			},
		}
		if block.HasChanges() {
			t.Error("HasChanges() = true、期待値 = false")
		}
	})

	t.Run("空のブロックはfalseを返す", func(t *testing.T) {
		t.Parallel()

		block := viewmodel.DiffBlock{}
		if block.HasChanges() {
			t.Error("HasChanges() = true、期待値 = false")
		}
	})
}

func TestNewBodyDiff_EmptyLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		oldText   string
		newText   string
		blankType viewmodel.DiffLineType
		oldNumber int
		newNumber int
	}{
		{
			name:    "空行を追加した後ろの変更行",
			oldText: "# 見出し\n同じ行\n*旧*\n", newText: "# 見出し\n\n同じ行\n*新*\n",
			blankType: viewmodel.DiffLineInsert, newNumber: 2,
		},
		{
			name:    "空行を削除した後ろの変更行",
			oldText: "# 見出し\n\n同じ行\n*旧*\n", newText: "# 見出し\n同じ行\n*新*\n",
			blankType: viewmodel.DiffLineDelete, oldNumber: 2,
		},
		{
			name:    "空の本文に空行1行を追加する",
			oldText: "", newText: "\n",
			blankType: viewmodel.DiffLineInsert, newNumber: 1,
		},
		{
			name:    "空行1行を削除して空の本文にする",
			oldText: "\n", newText: "",
			blankType: viewmodel.DiffLineDelete, oldNumber: 1,
		},
		{
			name:    "変更の無い空行の後ろの変更行",
			oldText: "\n同じ行\n*旧*\n", newText: "\n同じ行\n*新*\n",
			blankType: viewmodel.DiffLineEqual, oldNumber: 1, newNumber: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diff := viewmodel.NewBodyDiff(tt.oldText, tt.newText)
			if diff.Sources == nil {
				t.Fatal("新旧の本文が無い")
			}
			checkLine := func(source string, number int, content string) {
				t.Helper()
				lines := strings.Split(source, "\n")
				if number < 1 || number > len(lines) {
					t.Errorf("行番号 %d が本文の範囲外", number)
					return
				}
				if lines[number-1] != content {
					t.Errorf("本文の%d行目 = %q、差分の行の内容 = %q", number, lines[number-1], content)
				}
			}
			foundBlank := false
			for _, block := range diff.Blocks {
				for _, line := range block.Lines {
					if line.Type != viewmodel.DiffLineInsert {
						checkLine(diff.Sources.Old, line.OldNumber, line.Content)
					}
					if line.Type != viewmodel.DiffLineDelete {
						checkLine(diff.Sources.New, line.NewNumber, line.Content)
					}
					if line.Content == "" && line.Type == tt.blankType && line.OldNumber == tt.oldNumber && line.NewNumber == tt.newNumber {
						foundBlank = true
					}
				}
			}
			if !foundBlank {
				t.Errorf("空行の差分が無い: 種類=%v、旧行番号=%d、新行番号=%d", tt.blankType, tt.oldNumber, tt.newNumber)
			}
		})
	}
}

func TestNewBodyDiff(t *testing.T) {
	t.Parallel()

	t.Run("差分の計算に使った正規化後の新旧の本文を持つ", func(t *testing.T) {
		t.Parallel()

		diff := viewmodel.NewBodyDiff("# 見出し\r\n本文", "# 見出し\r\n**本文**\r\n")

		if len(diff.Blocks) == 0 {
			t.Fatal("Blocksが空")
		}
		if diff.Sources == nil {
			t.Fatal("Sourcesがnil")
		}
		if want := "# 見出し\n本文\n"; diff.Sources.Old != want {
			t.Errorf("Sources.Old = %q、期待値 = %q", diff.Sources.Old, want)
		}
		if want := "# 見出し\n**本文**\n"; diff.Sources.New != want {
			t.Errorf("Sources.New = %q、期待値 = %q", diff.Sources.New, want)
		}
	})

	t.Run("変更が無いときは本文を持たない", func(t *testing.T) {
		t.Parallel()

		diff := viewmodel.NewBodyDiff("本文\n", "本文\r\n")

		if len(diff.Blocks) != 0 {
			t.Errorf("Blocks = %v、期待値 = 空", diff.Blocks)
		}
		if diff.Sources != nil {
			t.Errorf("Sources = %v、期待値 = nil", diff.Sources)
		}
	})

	t.Run("新旧の本文の合計が上限ちょうどなら本文を持つ", func(t *testing.T) {
		t.Parallel()

		// 正規化で足される末尾の改行も数える
		oldText := strings.Repeat("a", viewmodel.DiffSourcesMaxBytes/2-1) + "\n"
		newText := strings.Repeat("b", viewmodel.DiffSourcesMaxBytes/2-1) + "\n"

		diff := viewmodel.NewBodyDiff(oldText, newText)

		if diff.Sources == nil {
			t.Error("Sourcesがnil")
		}
	})

	t.Run("新旧の本文の合計が上限を超えると本文を持たない", func(t *testing.T) {
		t.Parallel()

		// 和文はUTF-8で1文字3バイトのため、文字数ではなくバイト数で上限を超える
		oldText := strings.Repeat("あ", viewmodel.DiffSourcesMaxBytes/6+1) + "\n"
		newText := strings.Repeat("い", viewmodel.DiffSourcesMaxBytes/6+1) + "\n"

		diff := viewmodel.NewBodyDiff(oldText, newText)

		if len(diff.Blocks) == 0 {
			t.Error("Blocksが空")
		}
		if diff.Sources != nil {
			t.Error("上限を超えたのにSourcesを持っている")
		}
	})
}

// mismatchedDiffLinesは、差分の行のうち、内容がその行番号の指す新旧の本文の行と一致しないものを返す。
// ブラウザは行番号で本文の行を引いて記法の色を当てるため、一致しない行には色が付かない。
func mismatchedDiffLines(diff viewmodel.BodyDiff) []string {
	oldLines := strings.Split(diff.Sources.Old, "\n")
	newLines := strings.Split(diff.Sources.New, "\n")
	lineAt := func(lines []string, number int) (string, bool) {
		if number < 1 || number > len(lines) {
			return "", false
		}
		return lines[number-1], true
	}

	var mismatches []string
	for _, block := range diff.Blocks {
		for _, line := range block.Lines {
			if line.Type != viewmodel.DiffLineInsert {
				if source, ok := lineAt(oldLines, line.OldNumber); !ok || source != line.Content {
					mismatches = append(mismatches, fmt.Sprintf("旧の%d行目 = %q、差分の行の内容 = %q", line.OldNumber, source, line.Content))
				}
			}
			if line.Type != viewmodel.DiffLineDelete {
				if source, ok := lineAt(newLines, line.NewNumber); !ok || source != line.Content {
					mismatches = append(mismatches, fmt.Sprintf("新の%d行目 = %q、差分の行の内容 = %q", line.NewNumber, source, line.Content))
				}
			}
		}
	}
	return mismatches
}

func TestNewBodyDiff_LinesMatchSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		oldText string
		newText string
	}{
		{name: "空行に文字を書き足す", oldText: "\n- 項目\n", newText: "追記\n- 項目\n"},
		{name: "行の一部だけが前後の行と同じ文字の行を置き換える", oldText: "a\na\tb", newText: "b\n \n"},
		{name: "行末に書き足した行の前後で同じ行が続く", oldText: "本文です。\n本文です。\n以上です。\n", newText: "本文です。\n本文です。追記\n以上です。\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diff := viewmodel.NewBodyDiff(tt.oldText, tt.newText)
			if diff.Sources == nil {
				t.Fatal("新旧の本文が無い")
			}
			for _, mismatch := range mismatchedDiffLines(diff) {
				t.Error(mismatch)
			}
		})
	}

	t.Run("行の追加・削除・書き足しを組み合わせた多数の編集", func(t *testing.T) {
		t.Parallel()

		r := rand.New(rand.NewPCG(1, 2))
		pool := []string{"", "## 見出し", "- 項目", "本文です。", "**強調**です。", "> 引用", "a", "ab", "a b", "ba", " "}
		for i := range 2000 {
			// 異なる行が128を超えると、差分の計算で行を置き換える文字が複数バイトになるため、長い本文も含める。
			n := 1 + r.IntN(8)
			if i%10 == 0 {
				n = 150 + r.IntN(150)
			}
			lines := make([]string, n)
			for j := range lines {
				lines[j] = pool[r.IntN(len(pool))]
				if n > 100 {
					lines[j] += fmt.Sprintf("%d", r.IntN(n))
				}
			}
			edited := append([]string(nil), lines...)
			for range 1 + r.IntN(4) {
				k := r.IntN(len(edited) + 1)
				switch {
				case k == len(edited) || r.IntN(3) == 0:
					edited = append(edited[:k], append([]string{pool[r.IntN(len(pool))]}, edited[k:]...)...)
				case r.IntN(2) == 0:
					edited[k] += "追記"
				default:
					edited = append(edited[:k], edited[k+1:]...)
				}
			}
			oldText := strings.Join(lines, "\n") + "\n"
			newText := strings.Join(edited, "\n")

			diff := viewmodel.NewBodyDiff(oldText, newText)
			if diff.Sources == nil {
				continue
			}
			if mismatches := mismatchedDiffLines(diff); len(mismatches) > 0 {
				t.Fatalf("旧 = %q、新 = %q の差分で本文の行と一致しない: %v", oldText, newText, mismatches)
			}
		}
	})
}
