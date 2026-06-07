package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestDraftPageRevisionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRevisionRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draftpagerev-create@example.com").
		WithAtname("draftpagerevcreate").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draftpagerev-create").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		WithBodyHTML("<p>Draft body</p>").
		Build()

	t.Run("下書きページリビジョンを作成できる", func(t *testing.T) {
		revision, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Revision Title",
			Body:          "revision body",
			BodyHTML:      "<p>revision body</p>",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if revision == nil {
			t.Fatal("Create() returned nil, want draft page revision")
		}
		if revision.ID == "" {
			t.Error("revision.ID should not be empty")
		}
		if revision.DraftPageID != draftPageID {
			t.Errorf("revision.DraftPageID = %v, want %v", revision.DraftPageID, draftPageID)
		}
		if revision.SpaceID != spaceID {
			t.Errorf("revision.SpaceID = %v, want %v", revision.SpaceID, spaceID)
		}
		if revision.SpaceMemberID != spaceMemberID {
			t.Errorf("revision.SpaceMemberID = %v, want %v", revision.SpaceMemberID, spaceMemberID)
		}
		if revision.Title != "Revision Title" {
			t.Errorf("revision.Title = %v, want 'Revision Title'", revision.Title)
		}
		if revision.Body != "revision body" {
			t.Errorf("revision.Body = %v, want 'revision body'", revision.Body)
		}
		if revision.BodyHTML != "<p>revision body</p>" {
			t.Errorf("revision.BodyHTML = %v, want '<p>revision body</p>'", revision.BodyHTML)
		}
		if revision.CreatedAt.IsZero() {
			t.Error("revision.CreatedAt should not be zero")
		}
	})

	t.Run("下書きページIDに紐づくリビジョンをすべて削除できる", func(t *testing.T) {
		// リビジョンを2つ作成
		_, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Delete Test 1",
			Body:          "delete body 1",
			BodyHTML:      "<p>delete body 1</p>",
		})
		if err != nil {
			t.Fatalf("Create() first revision error = %v", err)
		}

		_, err = repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Delete Test 2",
			Body:          "delete body 2",
			BodyHTML:      "<p>delete body 2</p>",
		})
		if err != nil {
			t.Fatalf("Create() second revision error = %v", err)
		}

		// 削除
		err = repo.DeleteByDraftPageID(context.Background(), draftPageID, spaceID)
		if err != nil {
			t.Fatalf("DeleteByDraftPageID() error = %v", err)
		}
	})

	t.Run("下書きページIDに紐づくリビジョン件数を取得できる", func(t *testing.T) {
		// 検証用に独立した Page と DraftPage を作成 (unique 制約と他サブテストの干渉を避けるため)
		countPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Count Test Page").
			Build()

		countDraftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(countPageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("Count Draft Title").
			WithBody("count draft body").
			WithBodyHTML("<p>count draft body</p>").
			Build()

		// 0 件の状態で 0 が返ること
		count, err := repo.CountByDraftPageID(context.Background(), countDraftPageID, spaceID)
		if err != nil {
			t.Fatalf("CountByDraftPageID() error = %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}

		// 2 件作成 → 2 が返ること
		_, err = repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   countDraftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Count Test 1",
			Body:          "count body 1",
			BodyHTML:      "<p>count body 1</p>",
		})
		if err != nil {
			t.Fatalf("Create() first revision error = %v", err)
		}
		_, err = repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   countDraftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Count Test 2",
			Body:          "count body 2",
			BodyHTML:      "<p>count body 2</p>",
		})
		if err != nil {
			t.Fatalf("Create() second revision error = %v", err)
		}

		count, err = repo.CountByDraftPageID(context.Background(), countDraftPageID, spaceID)
		if err != nil {
			t.Fatalf("CountByDraftPageID() error = %v", err)
		}
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	// Verifies the ON DELETE CASCADE contract that the Rails-side deletion
	// paths rely on: deleting a draft_pages row directly must also delete
	// its revisions without an explicit DELETE on draft_page_revisions.
	//
	// [Ja] Rails 側の削除経路が頼る ON DELETE CASCADE の契約を検証する。
	// draft_pages の行を直接 DELETE したとき、draft_page_revisions への明示的な
	// DELETE なしでリビジョンも一緒に消えること。
	t.Run("下書きページの行を直接DELETEするとリビジョンも消える", func(t *testing.T) {
		// Create an independent Page and DraftPage for this verification (to avoid interference with other subtests)
		// [Ja] 検証用に独立した Page と DraftPage を作成 (他サブテストの干渉を避けるため)
		cascadePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(3).
			WithTitle("Cascade Test Page").
			Build()

		cascadeDraftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(cascadePageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("Cascade Draft Title").
			WithBody("cascade draft body").
			WithBodyHTML("<p>cascade draft body</p>").
			Build()

		_, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   cascadeDraftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Cascade Test",
			Body:          "cascade body",
			BodyHTML:      "<p>cascade body</p>",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Delete the parent draft_pages row directly (without going through application code)
		// [Ja] 親の draft_pages の行を直接削除 (アプリケーションコードを経由しない)
		_, err = tx.ExecContext(
			context.Background(),
			"DELETE FROM draft_pages WHERE id = $1 AND space_id = $2",
			string(cascadeDraftPageID), string(spaceID),
		)
		if err != nil {
			t.Fatalf("DELETE draft_pages error = %v", err)
		}

		count, err := repo.CountByDraftPageID(context.Background(), cascadeDraftPageID, spaceID)
		if err != nil {
			t.Fatalf("CountByDraftPageID() error = %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0 (revisions should be cascade-deleted)", count)
		}
	})

	t.Run("同じ下書きに対して複数のリビジョンを作成できる", func(t *testing.T) {
		revision1, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "First Revision",
			Body:          "first body",
			BodyHTML:      "<p>first body</p>",
		})
		if err != nil {
			t.Fatalf("Create() first revision error = %v", err)
		}

		revision2, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Second Revision",
			Body:          "second body",
			BodyHTML:      "<p>second body</p>",
		})
		if err != nil {
			t.Fatalf("Create() second revision error = %v", err)
		}

		if revision1.ID == revision2.ID {
			t.Errorf("revision1.ID and revision2.ID should be different, got %v", revision1.ID)
		}
		if revision1.Title != "First Revision" {
			t.Errorf("revision1.Title = %v, want 'First Revision'", revision1.Title)
		}
		if revision2.Title != "Second Revision" {
			t.Errorf("revision2.Title = %v, want 'Second Revision'", revision2.Title)
		}
	})
}
