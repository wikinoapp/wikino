package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestSuggestionPageRevisionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionPageRevisionRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spr-create@example.com").
		WithAtname("spr_create").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spr-create-space").
		WithName("SPR Create Space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0).
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
		WithTitle("テストページ").
		Build()

	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		WithTitle("テストページ").
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("テスト提案").
		Build()

	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		WithTitle("テスト提案ページ").
		Build()

	t.Run("タイトルありでリビジョンを作成できる", func(t *testing.T) {
		title := "リビジョンタイトル"
		rev, err := repo.Create(ctx, CreateSuggestionPageRevisionInput{
			SpaceID:             spaceID,
			SuggestionPageID:    suggestionPageID,
			EditorSpaceMemberID: spaceMemberID,
			Title:               &title,
			Body:                "リビジョン本文",
			BodyHTML:            "<p>リビジョン本文</p>",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if rev == nil {
			t.Fatal("Create() returned nil")
		}
		if rev.ID == "" {
			t.Error("rev.ID is empty")
		}
		if rev.SpaceID != spaceID {
			t.Errorf("rev.SpaceID = %v, want %v", rev.SpaceID, spaceID)
		}
		if rev.SuggestionPageID != suggestionPageID {
			t.Errorf("rev.SuggestionPageID = %v, want %v", rev.SuggestionPageID, suggestionPageID)
		}
		if rev.EditorSpaceMemberID != spaceMemberID {
			t.Errorf("rev.EditorSpaceMemberID = %v, want %v", rev.EditorSpaceMemberID, spaceMemberID)
		}
		if rev.Title == nil || *rev.Title != "リビジョンタイトル" {
			t.Errorf("rev.Title = %v, want リビジョンタイトル", rev.Title)
		}
		if rev.Body != "リビジョン本文" {
			t.Errorf("rev.Body = %v, want リビジョン本文", rev.Body)
		}
		if rev.BodyHTML != "<p>リビジョン本文</p>" {
			t.Errorf("rev.BodyHTML = %v, want <p>リビジョン本文</p>", rev.BodyHTML)
		}
		if rev.CreatedAt.IsZero() {
			t.Error("rev.CreatedAt is zero")
		}
		if rev.UpdatedAt.IsZero() {
			t.Error("rev.UpdatedAt is zero")
		}
	})

	t.Run("タイトルなしでリビジョンを作成できる", func(t *testing.T) {
		rev, err := repo.Create(ctx, CreateSuggestionPageRevisionInput{
			SpaceID:             spaceID,
			SuggestionPageID:    suggestionPageID,
			EditorSpaceMemberID: spaceMemberID,
			Title:               nil,
			Body:                "タイトルなし本文",
			BodyHTML:            "<p>タイトルなし本文</p>",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if rev.Title != nil {
			t.Errorf("rev.Title = %v, want nil", rev.Title)
		}
	})
}

func TestSuggestionPageRevisionRepository_ListBySuggestionPageID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionPageRevisionRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spr-list@example.com").
		WithAtname("spr_list").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spr-list-space").
		WithName("SPR List Space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0).
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
		Build()

	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	testutil.NewSuggestionPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionPageID(suggestionPageID).
		WithEditorSpaceMemberID(spaceMemberID).
		WithTitle("リビジョン1").
		Build()

	// 作成日時の順序を確保するため少し待つ
	time.Sleep(time.Millisecond)

	testutil.NewSuggestionPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionPageID(suggestionPageID).
		WithEditorSpaceMemberID(spaceMemberID).
		WithTitle("リビジョン2").
		Build()

	t.Run("編集提案ページに紐づくリビジョン一覧を取得できる", func(t *testing.T) {
		revisions, err := repo.ListBySuggestionPageID(ctx, suggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionPageID() error = %v", err)
		}
		if len(revisions) != 2 {
			t.Fatalf("len(revisions) = %v, want 2", len(revisions))
		}
		if revisions[0].Title == nil || *revisions[0].Title != "リビジョン1" {
			t.Errorf("revisions[0].Title = %v, want リビジョン1", revisions[0].Title)
		}
		if revisions[1].Title == nil || *revisions[1].Title != "リビジョン2" {
			t.Errorf("revisions[1].Title = %v, want リビジョン2", revisions[1].Title)
		}
	})

	t.Run("該当なしの場合は空のスライスを返す", func(t *testing.T) {
		// 別のSuggestionPageを作成（リビジョンなし）
		pageID2 := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("リスト空テスト用ページ").
			Build()

		pageRevisionID2 := testutil.NewPageRevisionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID2).
			Build()

		emptySuggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID2).
			WithPageRevisionID(pageRevisionID2).
			Build()

		revisions, err := repo.ListBySuggestionPageID(ctx, emptySuggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionPageID() error = %v", err)
		}
		if len(revisions) != 0 {
			t.Errorf("len(revisions) = %v, want 0", len(revisions))
		}
	})
}

func TestSuggestionPageRevisionRepository_FindLatest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionPageRevisionRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spr-latest@example.com").
		WithAtname("spr_latest").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spr-latest-space").
		WithName("SPR Latest Space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0).
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
		Build()

	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	testutil.NewSuggestionPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionPageID(suggestionPageID).
		WithEditorSpaceMemberID(spaceMemberID).
		WithTitle("古いリビジョン").
		Build()

	time.Sleep(time.Millisecond)

	testutil.NewSuggestionPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionPageID(suggestionPageID).
		WithEditorSpaceMemberID(spaceMemberID).
		WithTitle("最新リビジョン").
		Build()

	t.Run("最新のリビジョンを取得できる", func(t *testing.T) {
		rev, err := repo.FindLatest(ctx, suggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("FindLatest() error = %v", err)
		}
		if rev == nil {
			t.Fatal("FindLatest() returned nil")
		}
		if rev.Title == nil || *rev.Title != "最新リビジョン" {
			t.Errorf("rev.Title = %v, want 最新リビジョン", rev.Title)
		}
	})

	t.Run("リビジョンがない場合はnilを返す", func(t *testing.T) {
		pageID2 := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("最新空テスト用ページ").
			Build()

		pageRevisionID2 := testutil.NewPageRevisionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID2).
			Build()

		emptySuggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID2).
			WithPageRevisionID(pageRevisionID2).
			Build()

		rev, err := repo.FindLatest(ctx, emptySuggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("FindLatest() error = %v", err)
		}
		if rev != nil {
			t.Errorf("FindLatest() = %v, want nil", rev)
		}
	})

	t.Run("異なるスペースIDではnilを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("spr-latest-other").
			WithName("Other Space").
			Build()

		rev, err := repo.FindLatest(ctx, suggestionPageID, otherSpaceID)
		if err != nil {
			t.Fatalf("FindLatest() error = %v", err)
		}
		if rev != nil {
			t.Errorf("FindLatest() = %v, want nil", rev)
		}
	})
}
