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
			t.Errorf("len(blocks) = %d, want 0", len(blocks))
		}
	})

	t.Run("空テキスト同士の場合は空を返す", func(t *testing.T) {
		t.Parallel()

		blocks := viewmodel.ComputeDiffBlocks("", "", 3)
		if len(blocks) != 0 {
			t.Errorf("len(blocks) = %d, want 0", len(blocks))
		}
	})

	t.Run("行の追加を検出する", func(t *testing.T) {
		t.Parallel()

		oldText := "line1\nline2\n"
		newText := "line1\nline2\nline3\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("expected at least one block")
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
			t.Error("expected an insert line with content 'line3'")
		}
	})

	t.Run("行の削除を検出する", func(t *testing.T) {
		t.Parallel()

		oldText := "line1\nline2\nline3\n"
		newText := "line1\nline3\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("expected at least one block")
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
			t.Error("expected a delete line with content 'line2'")
		}
	})

	t.Run("行の変更を検出する", func(t *testing.T) {
		t.Parallel()

		oldText := "line1\nold content\nline3\n"
		newText := "line1\nnew content\nline3\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("expected at least one block")
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
			t.Error("expected a delete line with content 'old content'")
		}
		if !hasInsert {
			t.Error("expected an insert line with content 'new content'")
		}
	})

	t.Run("行番号が正しく設定される", func(t *testing.T) {
		t.Parallel()

		oldText := "line1\nline2\nline3\n"
		newText := "line1\nnew line\nline2\nline3\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) == 0 {
			t.Fatal("expected at least one block")
		}

		for _, block := range blocks {
			for _, line := range block.Lines {
				switch line.Type {
				case viewmodel.DiffLineEqual:
					if line.OldNumber == 0 {
						t.Errorf("equal line should have OldNumber > 0, got %d", line.OldNumber)
					}
					if line.NewNumber == 0 {
						t.Errorf("equal line should have NewNumber > 0, got %d", line.NewNumber)
					}
				case viewmodel.DiffLineDelete:
					if line.OldNumber == 0 {
						t.Errorf("delete line should have OldNumber > 0, got %d", line.OldNumber)
					}
				case viewmodel.DiffLineInsert:
					if line.NewNumber == 0 {
						t.Errorf("insert line should have NewNumber > 0, got %d", line.NewNumber)
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
			t.Errorf("len(blocks) = %d, want 2", len(blocks))
		}
	})

	t.Run("近い変更はマージされる", func(t *testing.T) {
		t.Parallel()

		// 行2と行4を変更。contextLines=3なので1つのブロックにマージされる
		oldText := "line1\nline2\nline3\nline4\nline5\n"
		newText := "line1\nchanged2\nline3\nchanged4\nline5\n"
		blocks := viewmodel.ComputeDiffBlocks(oldText, newText, 3)

		if len(blocks) != 1 {
			t.Errorf("len(blocks) = %d, want 1", len(blocks))
		}
	})

	t.Run("空テキストから新規テキストへの差分", func(t *testing.T) {
		t.Parallel()

		blocks := viewmodel.ComputeDiffBlocks("", "new line\n", 3)

		if len(blocks) == 0 {
			t.Fatal("expected at least one block")
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
			t.Error("expected insert lines")
		}
	})

	t.Run("テキストから空テキストへの差分", func(t *testing.T) {
		t.Parallel()

		blocks := viewmodel.ComputeDiffBlocks("old line\n", "", 3)

		if len(blocks) == 0 {
			t.Fatal("expected at least one block")
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
			t.Error("expected delete lines")
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
			t.Error("HasChanges() = false, want true")
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
			t.Error("HasChanges() = true, want false")
		}
	})

	t.Run("空のブロックはfalseを返す", func(t *testing.T) {
		t.Parallel()

		block := viewmodel.DiffBlock{}
		if block.HasChanges() {
			t.Error("HasChanges() = true, want false")
		}
	})
}
