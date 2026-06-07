package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestSuggestionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("suggestion-create@example.com").
		WithAtname("suggestion_create").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("suggestion-create-space").
		WithName("Suggestion Create Space").
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

	t.Run("編集提案を作成できる", func(t *testing.T) {
		suggestion, err := repo.Create(ctx, CreateSuggestionInput{
			SpaceID:              spaceID,
			TopicID:              topicID,
			CreatedSpaceMemberID: spaceMemberID,
			Title:                "テスト提案",
			Body:                 "テスト本文",
			Status:               model.SuggestionStatusDraft,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if suggestion == nil {
			t.Fatal("Create() returned nil")
		}
		if suggestion.ID == "" {
			t.Error("suggestion.ID is empty")
		}
		if suggestion.SpaceID != spaceID {
			t.Errorf("suggestion.SpaceID = %v, want %v", suggestion.SpaceID, spaceID)
		}
		if suggestion.TopicID != topicID {
			t.Errorf("suggestion.TopicID = %v, want %v", suggestion.TopicID, topicID)
		}
		if suggestion.CreatedSpaceMemberID != spaceMemberID {
			t.Errorf("suggestion.CreatedSpaceMemberID = %v, want %v", suggestion.CreatedSpaceMemberID, spaceMemberID)
		}
		if suggestion.Title != "テスト提案" {
			t.Errorf("suggestion.Title = %v, want テスト提案", suggestion.Title)
		}
		if suggestion.Body != "テスト本文" {
			t.Errorf("suggestion.Body = %v, want テスト本文", suggestion.Body)
		}
		if suggestion.Status != model.SuggestionStatusDraft {
			t.Errorf("suggestion.Status = %v, want %v", suggestion.Status, model.SuggestionStatusDraft)
		}
		if suggestion.AppliedAt != nil {
			t.Errorf("suggestion.AppliedAt = %v, want nil", suggestion.AppliedAt)
		}
	})
}

func TestSuggestionRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("suggestion-find@example.com").
		WithAtname("suggestion_find").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("suggestion-find-space").
		WithName("Suggestion Find Space").
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

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("検索テスト提案").
		Build()

	t.Run("IDで編集提案を取得できる", func(t *testing.T) {
		suggestion, err := repo.FindByID(ctx, suggestionID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if suggestion == nil {
			t.Fatal("FindByID() returned nil")
		}
		if suggestion.ID != suggestionID {
			t.Errorf("suggestion.ID = %v, want %v", suggestion.ID, suggestionID)
		}
		if suggestion.Title != "検索テスト提案" {
			t.Errorf("suggestion.Title = %v, want 検索テスト提案", suggestion.Title)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		suggestion, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000", spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if suggestion != nil {
			t.Errorf("FindByID() = %v, want nil", suggestion)
		}
	})

	t.Run("異なるスペースIDではnilを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("suggestion-find-other").
			WithName("Other Space").
			Build()

		suggestion, err := repo.FindByID(ctx, suggestionID, otherSpaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if suggestion != nil {
			t.Errorf("FindByID() = %v, want nil", suggestion)
		}
	})
}

func TestSuggestionRepository_ListByTopicAndStatuses(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("suggestion-list@example.com").
		WithAtname("suggestion_list").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("suggestion-list-space").
		WithName("Suggestion List Space").
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

	// 各ステータスの編集提案を作成
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("下書き提案").
		WithStatus(model.SuggestionStatusDraft).
		Build()

	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("オープン提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("反映済み提案").
		WithStatus(model.SuggestionStatusApplied).
		Build()

	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("クローズ提案").
		WithStatus(model.SuggestionStatusClosed).
		Build()

	t.Run("オープンステータスの編集提案を取得できる", func(t *testing.T) {
		suggestions, err := repo.ListByTopicAndStatuses(ctx, topicID, spaceID, []model.SuggestionStatus{model.SuggestionStatusDraft, model.SuggestionStatusOpen})
		if err != nil {
			t.Fatalf("ListByTopicAndStatuses() error = %v", err)
		}
		if len(suggestions) != 2 {
			t.Fatalf("len(suggestions) = %v, want 2", len(suggestions))
		}
	})

	t.Run("クローズステータスの編集提案を取得できる", func(t *testing.T) {
		suggestions, err := repo.ListByTopicAndStatuses(ctx, topicID, spaceID, []model.SuggestionStatus{model.SuggestionStatusApplied, model.SuggestionStatusClosed})
		if err != nil {
			t.Fatalf("ListByTopicAndStatuses() error = %v", err)
		}
		if len(suggestions) != 2 {
			t.Fatalf("len(suggestions) = %v, want 2", len(suggestions))
		}
	})

	t.Run("該当なしの場合は空のスライスを返す", func(t *testing.T) {
		otherTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("Empty Topic").
			Build()

		suggestions, err := repo.ListByTopicAndStatuses(ctx, otherTopicID, spaceID, []model.SuggestionStatus{model.SuggestionStatusOpen})
		if err != nil {
			t.Fatalf("ListByTopicAndStatuses() error = %v", err)
		}
		if len(suggestions) != 0 {
			t.Errorf("len(suggestions) = %v, want 0", len(suggestions))
		}
	})
}

func TestSuggestionRepository_UpdateStatus(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("suggestion-update@example.com").
		WithAtname("suggestion_update").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("suggestion-update-space").
		WithName("Suggestion Update Space").
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

	t.Run("ステータスをオープンに更新できる", func(t *testing.T) {
		suggestionID := testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("ステータス更新テスト").
			WithStatus(model.SuggestionStatusDraft).
			Build()

		suggestion, err := repo.UpdateStatus(ctx, UpdateStatusInput{
			ID:      suggestionID,
			SpaceID: spaceID,
			Status:  model.SuggestionStatusOpen,
		})
		if err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}
		if suggestion.Status != model.SuggestionStatusOpen {
			t.Errorf("suggestion.Status = %v, want %v", suggestion.Status, model.SuggestionStatusOpen)
		}
		if suggestion.AppliedAt != nil {
			t.Errorf("suggestion.AppliedAt = %v, want nil", suggestion.AppliedAt)
		}
	})

	t.Run("ステータスを反映済みに更新しapplied_atを設定できる", func(t *testing.T) {
		suggestionID := testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("反映テスト").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		appliedAt := time.Now()
		suggestion, err := repo.UpdateStatus(ctx, UpdateStatusInput{
			ID:        suggestionID,
			SpaceID:   spaceID,
			Status:    model.SuggestionStatusApplied,
			AppliedAt: &appliedAt,
		})
		if err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}
		if suggestion.Status != model.SuggestionStatusApplied {
			t.Errorf("suggestion.Status = %v, want %v", suggestion.Status, model.SuggestionStatusApplied)
		}
		if suggestion.AppliedAt == nil {
			t.Fatal("suggestion.AppliedAt is nil, want non-nil")
		}
	})
}

func TestSuggestionRepository_CountByTopicAndStatuses(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("suggestion-count@example.com").
		WithAtname("suggestion_count").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("suggestion-count-space").
		WithName("Suggestion Count Space").
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

	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusDraft).
		Build()

	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("オープン提案1").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("オープン提案2").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	t.Run("オープンステータスの件数を取得できる", func(t *testing.T) {
		count, err := repo.CountByTopicAndStatuses(ctx, topicID, spaceID, []model.SuggestionStatus{model.SuggestionStatusOpen})
		if err != nil {
			t.Fatalf("CountByTopicAndStatuses() error = %v", err)
		}
		if count != 2 {
			t.Errorf("count = %v, want 2", count)
		}
	})

	t.Run("複数ステータスの件数を取得できる", func(t *testing.T) {
		count, err := repo.CountByTopicAndStatuses(ctx, topicID, spaceID, []model.SuggestionStatus{model.SuggestionStatusDraft, model.SuggestionStatusOpen})
		if err != nil {
			t.Fatalf("CountByTopicAndStatuses() error = %v", err)
		}
		if count != 3 {
			t.Errorf("count = %v, want 3", count)
		}
	})

	t.Run("該当なしの場合は0を返す", func(t *testing.T) {
		count, err := repo.CountByTopicAndStatuses(ctx, topicID, spaceID, []model.SuggestionStatus{model.SuggestionStatusClosed})
		if err != nil {
			t.Fatalf("CountByTopicAndStatuses() error = %v", err)
		}
		if count != 0 {
			t.Errorf("count = %v, want 0", count)
		}
	})
}

// Verifies the ON DELETE CASCADE contract that the Rails-side deletion paths
// (e.g. DestroyTopicJob) rely on: deleting a topics row directly must also
// delete the whole suggestion tree (suggestions -> suggestion_pages ->
// suggestion_page_revisions / suggestion_comments) without explicit DELETEs.
//
// [Ja] Rails 側の削除経路 (DestroyTopicJob など) が頼る ON DELETE CASCADE の
// 契約を検証する。topics の行を直接 DELETE したとき、明示的な DELETE なしで
// 提案ツリー全体 (suggestions -> suggestion_pages -> suggestion_page_revisions
// / suggestion_comments) も一緒に消えること。
func TestSuggestionRepository_CascadeOnTopicDelete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("suggestion-cascade@example.com").
		WithAtname("suggestion_cascade").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("suggestion-cascade").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	// The deletion target topic. The page lives in another topic so that the
	// pages.topic_id FK (Rails-era, no cascade) does not block the DELETE.
	// [Ja] 削除対象のトピック。pages.topic_id の FK (Rails 由来、CASCADE なし)
	// が DELETE を妨げないよう、ページは別トピックに置く。
	targetTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Target Topic").
		Build()

	otherTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Other Topic").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(otherTopicID).
		WithNumber(1).
		WithTitle("Suggested Page").
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(targetTopicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		Build()

	suggestionPageRevisionID := testutil.NewSuggestionPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionPageID(suggestionPageID).
		WithEditorSpaceMemberID(spaceMemberID).
		Build()

	suggestionCommentID := testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	t.Run("トピックの行を直接DELETEすると提案ツリーも消える", func(t *testing.T) {
		// Delete the parent topics row directly (without going through application code)
		// [Ja] 親の topics の行を直接削除 (アプリケーションコードを経由しない)
		_, err := tx.ExecContext(
			ctx,
			"DELETE FROM topics WHERE id = $1 AND space_id = $2",
			string(targetTopicID), string(spaceID),
		)
		if err != nil {
			t.Fatalf("DELETE topics error = %v", err)
		}

		counts := []struct {
			table string
			id    string
		}{
			{"suggestions", string(suggestionID)},
			{"suggestion_pages", string(suggestionPageID)},
			{"suggestion_page_revisions", string(suggestionPageRevisionID)},
			{"suggestion_comments", string(suggestionCommentID)},
		}
		for _, c := range counts {
			var count int
			err := tx.QueryRowContext(
				ctx,
				"SELECT COUNT(*) FROM "+c.table+" WHERE id = $1 AND space_id = $2",
				c.id, string(spaceID),
			).Scan(&count)
			if err != nil {
				t.Fatalf("SELECT COUNT(*) FROM %s error = %v", c.table, err)
			}
			if count != 0 {
				t.Errorf("%s count = %d, want 0 (should be cascade-deleted)", c.table, count)
			}
		}

		// The page in the other topic must survive the cascade.
		// [Ja] 別トピックのページは連鎖削除に巻き込まれないこと。
		var pageCount int
		if err := tx.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM pages WHERE id = $1 AND space_id = $2",
			string(pageID), string(spaceID),
		).Scan(&pageCount); err != nil {
			t.Fatalf("SELECT COUNT(*) FROM pages error = %v", err)
		}
		if pageCount != 1 {
			t.Errorf("pages count = %d, want 1 (page must not be deleted)", pageCount)
		}
	})
}
