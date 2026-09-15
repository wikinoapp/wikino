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

	// newRevisionはビューモデル変換テスト用の最小限のリビジョンモデルを生成する。
	newRevision := func(createdAt time.Time) *model.DraftPageRevision {
		return &model.DraftPageRevision{
			ID:        model.DraftPageRevisionID("00000000-0000-0000-0000-000000000000"),
			CreatedAt: createdAt,
		}
	}

	t.Run("新しい順のスライスからバージョン番号を算出する", func(t *testing.T) {
		t.Parallel()

		// 新しい順: インデックス0が最新リビジョン (v3)。
		revisions := []*model.DraftPageRevision{
			newRevision(base.Add(3 * time.Second)),
			newRevision(base.Add(2 * time.Second)),
			newRevision(base.Add(1 * time.Second)),
		}

		result := viewmodel.NewDraftPageRevisions(revisions, 3)

		if len(result) != 3 {
			t.Fatalf("len(result) = %d、期待値 = 3", len(result))
		}
		wantVersions := []int64{3, 2, 1}
		for i, want := range wantVersions {
			if result[i].Version != want {
				t.Errorf("result[%d].Version = %d、期待値 = %d", i, result[i].Version, want)
			}
		}
		if !result[0].IsCurrent {
			t.Error("result[0].IsCurrentがfalse (最新のリビジョン)")
		}
		for i := 1; i < len(result); i++ {
			if result[i].IsCurrent {
				t.Errorf("result[%d].IsCurrentがtrue", i)
			}
		}
		if !result[0].CreatedAt.Equal(base.Add(3 * time.Second)) {
			t.Errorf("result[0].CreatedAt = %v、期待値 = %v", result[0].CreatedAt, base.Add(3*time.Second))
		}
		if result[0].ID != "00000000-0000-0000-0000-000000000000" {
			t.Errorf("result[0].ID = %q、期待値 = %q", result[0].ID, "00000000-0000-0000-0000-000000000000")
		}
	})

	t.Run("総件数が一覧の件数より多い場合もバージョン番号が安定する", func(t *testing.T) {
		t.Parallel()

		// 総件数25件で一覧が2件にキャップされた場合: バージョンはv25とv24になる
		// (上限から溢れた古いリビジョンがv1〜v23を保持する)。
		revisions := []*model.DraftPageRevision{
			newRevision(base.Add(25 * time.Second)),
			newRevision(base.Add(24 * time.Second)),
		}

		result := viewmodel.NewDraftPageRevisions(revisions, 25)

		if len(result) != 2 {
			t.Fatalf("len(result) = %d、期待値 = 2", len(result))
		}
		if result[0].Version != 25 {
			t.Errorf("result[0].Version = %d、期待値 = 25", result[0].Version)
		}
		if result[1].Version != 24 {
			t.Errorf("result[1].Version = %d、期待値 = 24", result[1].Version)
		}
	})

	t.Run("空のスライスでは空の一覧を返す", func(t *testing.T) {
		t.Parallel()

		result := viewmodel.NewDraftPageRevisions(nil, 0)

		if result == nil {
			t.Fatal("resultがnil (期待値 = nilではない空のスライス)")
		}
		if len(result) != 0 {
			t.Errorf("len(result) = %d、期待値 = 0", len(result))
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
			t.Error("HasTitleChangeがfalse")
		}
		if diff.OldTitle != "Old Title" || diff.NewTitle != "New Title" {
			t.Errorf("OldTitle/NewTitle = %q/%q、期待値 = %q/%q", diff.OldTitle, diff.NewTitle, "Old Title", "New Title")
		}
		if !diff.CreatedAt.Equal(createdAt) {
			t.Errorf("CreatedAt = %v、期待値 = %v", diff.CreatedAt, createdAt)
		}
		if len(diff.BodyBlocks) == 0 {
			t.Fatal("BodyBlocksが空")
		}
		// 追加された行のみが挿入行として現れること。
		var inserts int
		for _, block := range diff.BodyBlocks {
			for _, line := range block.Lines {
				if line.Type == viewmodel.DiffLineInsert {
					inserts++
					if line.Content != "line two" {
						t.Errorf("追加行 = %q、期待値 = %q", line.Content, "line two")
					}
				}
				if line.Type == viewmodel.DiffLineDelete {
					t.Errorf("予期しない削除行: %q", line.Content)
				}
			}
		}
		if inserts != 1 {
			t.Errorf("inserts = %d、期待値 = 1", inserts)
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
			t.Error("HasTitleChangeがfalse (空から空以外への変更)")
		}
		if diff.OldTitle != "" {
			t.Errorf("OldTitle = %q、期待値 = 空", diff.OldTitle)
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
			t.Errorf("追加行/その他 = %d/%d、期待値 = 2/0", inserts, others)
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
			t.Error("HasTitleChangeがtrue")
		}
		if len(diff.BodyBlocks) != 0 {
			t.Errorf("BodyBlocks = %v、期待値 = 空", diff.BodyBlocks)
		}
	})
}
