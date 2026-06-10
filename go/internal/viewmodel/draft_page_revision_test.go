package viewmodel_test

import (
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewDraftPageRevisions(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// newRevision builds a minimal revision model for view-model conversion tests.
	// [Ja] newRevision はビューモデル変換テスト用の最小限のリビジョンモデルを生成する。
	newRevision := func(createdAt time.Time) *model.DraftPageRevision {
		return &model.DraftPageRevision{
			ID:        model.DraftPageRevisionID("00000000-0000-0000-0000-000000000000"),
			CreatedAt: createdAt,
		}
	}

	t.Run("新しい順のスライスからバージョン番号を算出する", func(t *testing.T) {
		t.Parallel()

		// Newest first: index 0 is the latest revision (v3).
		// [Ja] 新しい順: インデックス 0 が最新リビジョン (v3)。
		revisions := []*model.DraftPageRevision{
			newRevision(base.Add(3 * time.Second)),
			newRevision(base.Add(2 * time.Second)),
			newRevision(base.Add(1 * time.Second)),
		}

		result := viewmodel.NewDraftPageRevisions(revisions, 3)

		if len(result) != 3 {
			t.Fatalf("len(result) = %d, want 3", len(result))
		}
		wantVersions := []int64{3, 2, 1}
		for i, want := range wantVersions {
			if result[i].Version != want {
				t.Errorf("result[%d].Version = %d, want %d", i, result[i].Version, want)
			}
		}
		if !result[0].IsCurrent {
			t.Error("result[0].IsCurrent should be true (newest revision)")
		}
		for i := 1; i < len(result); i++ {
			if result[i].IsCurrent {
				t.Errorf("result[%d].IsCurrent should be false", i)
			}
		}
		if !result[0].CreatedAt.Equal(base.Add(3 * time.Second)) {
			t.Errorf("result[0].CreatedAt = %v, want %v", result[0].CreatedAt, base.Add(3*time.Second))
		}
		if result[0].ID != "00000000-0000-0000-0000-000000000000" {
			t.Errorf("result[0].ID = %q, want %q", result[0].ID, "00000000-0000-0000-0000-000000000000")
		}
	})

	t.Run("総件数が一覧の件数より多い場合もバージョン番号が安定する", func(t *testing.T) {
		t.Parallel()

		// Total count 25 with a list capped at 2 entries: versions are v25 and v24
		// (older revisions outside the cap keep v1..v23).
		//
		// [Ja] 総件数 25 件で一覧が 2 件にキャップされた場合: バージョンは v25 と v24 になる
		// (上限から溢れた古いリビジョンが v1〜v23 を保持する)。
		revisions := []*model.DraftPageRevision{
			newRevision(base.Add(25 * time.Second)),
			newRevision(base.Add(24 * time.Second)),
		}

		result := viewmodel.NewDraftPageRevisions(revisions, 25)

		if len(result) != 2 {
			t.Fatalf("len(result) = %d, want 2", len(result))
		}
		if result[0].Version != 25 {
			t.Errorf("result[0].Version = %d, want 25", result[0].Version)
		}
		if result[1].Version != 24 {
			t.Errorf("result[1].Version = %d, want 24", result[1].Version)
		}
	})

	t.Run("空のスライスでは空の一覧を返す", func(t *testing.T) {
		t.Parallel()

		result := viewmodel.NewDraftPageRevisions(nil, 0)

		if result == nil {
			t.Fatal("result should be non-nil empty slice")
		}
		if len(result) != 0 {
			t.Errorf("len(result) = %d, want 0", len(result))
		}
	})
}

func TestNewDraftPageRevisionDiff(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("直前リビジョンとの差分を計算する", func(t *testing.T) {
		t.Parallel()

		previous := &model.DraftPageRevision{
			Title:     "Old Title",
			Body:      "line one\n",
			CreatedAt: createdAt.Add(-time.Minute),
		}
		revision := &model.DraftPageRevision{
			Title:     "New Title",
			Body:      "line one\nline two\n",
			CreatedAt: createdAt,
		}

		diff := viewmodel.NewDraftPageRevisionDiff(revision, previous)

		if !diff.HasTitleChange {
			t.Error("HasTitleChange should be true")
		}
		if diff.OldTitle != "Old Title" || diff.NewTitle != "New Title" {
			t.Errorf("OldTitle/NewTitle = %q/%q, want %q/%q", diff.OldTitle, diff.NewTitle, "Old Title", "New Title")
		}
		if !diff.CreatedAt.Equal(createdAt) {
			t.Errorf("CreatedAt = %v, want %v", diff.CreatedAt, createdAt)
		}
		if len(diff.BodyBlocks) == 0 {
			t.Fatal("BodyBlocks should not be empty")
		}
		// Only the added line should appear as an insert.
		// [Ja] 追加された行のみが挿入行として現れること。
		var inserts int
		for _, block := range diff.BodyBlocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineInsert {
					inserts++
					if line.Content != "line two" {
						t.Errorf("insert line = %q, want %q", line.Content, "line two")
					}
				}
				if line.Type == viewmodel.DiffLineDelete {
					t.Errorf("unexpected delete line: %q", line.Content)
				}
			}
		}
		if inserts != 1 {
			t.Errorf("inserts = %d, want 1", inserts)
		}
	})

	t.Run("直前リビジョンがない場合は全文追加になる", func(t *testing.T) {
		t.Parallel()

		revision := &model.DraftPageRevision{
			Title:     "First Title",
			Body:      "line one\nline two\n",
			CreatedAt: createdAt,
		}

		diff := viewmodel.NewDraftPageRevisionDiff(revision, nil)

		if !diff.HasTitleChange {
			t.Error("HasTitleChange should be true (empty -> non-empty)")
		}
		if diff.OldTitle != "" {
			t.Errorf("OldTitle = %q, want empty", diff.OldTitle)
		}
		var inserts, others int
		for _, block := range diff.BodyBlocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineInsert {
					inserts++
				} else {
					others++
				}
			}
		}
		if inserts != 2 || others != 0 {
			t.Errorf("inserts/others = %d/%d, want 2/0", inserts, others)
		}
	})

	t.Run("内容が同一なら差分なしになる", func(t *testing.T) {
		t.Parallel()

		previous := &model.DraftPageRevision{
			Title:     "Same Title",
			Body:      "same body\n",
			CreatedAt: createdAt.Add(-time.Minute),
		}
		revision := &model.DraftPageRevision{
			Title:     "Same Title",
			Body:      "same body\n",
			CreatedAt: createdAt,
		}

		diff := viewmodel.NewDraftPageRevisionDiff(revision, previous)

		if diff.HasTitleChange {
			t.Error("HasTitleChange should be false")
		}
		if len(diff.BodyBlocks) != 0 {
			t.Errorf("BodyBlocks = %v, want empty", diff.BodyBlocks)
		}
	})
}
