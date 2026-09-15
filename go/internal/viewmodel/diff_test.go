package viewmodel_test

import (
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
