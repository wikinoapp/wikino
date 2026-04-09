package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestSuggestionCommentRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionCommentRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sc-create@example.com").
		WithAtname("sc_create").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sc-create-space").
		WithName("SC Create Space").
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

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("テスト提案").
		Build()

	t.Run("コメントを作成できる", func(t *testing.T) {
		comment, err := repo.Create(ctx, CreateSuggestionCommentInput{
			SpaceID:              spaceID,
			SuggestionID:         suggestionID,
			CreatedSpaceMemberID: spaceMemberID,
			Number:               1,
			Body:                 "コメント本文",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if comment == nil {
			t.Fatal("Create() returned nil")
		}
		if comment.ID == "" {
			t.Error("comment.ID is empty")
		}
		if comment.SpaceID != spaceID {
			t.Errorf("comment.SpaceID = %v, want %v", comment.SpaceID, spaceID)
		}
		if comment.SuggestionID != suggestionID {
			t.Errorf("comment.SuggestionID = %v, want %v", comment.SuggestionID, suggestionID)
		}
		if comment.CreatedSpaceMemberID != spaceMemberID {
			t.Errorf("comment.CreatedSpaceMemberID = %v, want %v", comment.CreatedSpaceMemberID, spaceMemberID)
		}
		if comment.Number != 1 {
			t.Errorf("comment.Number = %v, want 1", comment.Number)
		}
		if comment.Body != "コメント本文" {
			t.Errorf("comment.Body = %v, want コメント本文", comment.Body)
		}
		if comment.CreatedAt.IsZero() {
			t.Error("comment.CreatedAt is zero")
		}
		if comment.UpdatedAt.IsZero() {
			t.Error("comment.UpdatedAt is zero")
		}
	})
}

func TestSuggestionCommentRepository_GetNextNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionCommentRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sc-nextnumber@example.com").
		WithAtname("sc_nextnumber").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sc-nextnumber-space").
		WithName("SC NextNumber Space").
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

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("番号テスト提案").
		Build()

	t.Run("コメントがない場合は1を返す", func(t *testing.T) {
		n, err := repo.GetNextNumber(ctx, suggestionID)
		if err != nil {
			t.Fatalf("GetNextNumber() error = %v", err)
		}
		if n != 1 {
			t.Errorf("GetNextNumber() = %v, want 1", n)
		}
	})

	t.Run("コメントがある場合は次の番号を返す", func(t *testing.T) {
		testutil.NewSuggestionCommentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithCreatedSpaceMemberID(spaceMemberID).
			Build()

		n, err := repo.GetNextNumber(ctx, suggestionID)
		if err != nil {
			t.Fatalf("GetNextNumber() error = %v", err)
		}
		if n != 2 {
			t.Errorf("GetNextNumber() = %v, want 2", n)
		}
	})
}

func TestSuggestionCommentRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionCommentRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sc-find@example.com").
		WithAtname("sc_find").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sc-find-space").
		WithName("SC Find Space").
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

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	commentID := testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("検索対象コメント").
		Build()

	t.Run("IDでコメントを取得できる", func(t *testing.T) {
		comment, err := repo.FindByID(ctx, commentID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if comment == nil {
			t.Fatal("FindByID() returned nil")
		}
		if comment.ID != commentID {
			t.Errorf("comment.ID = %v, want %v", comment.ID, commentID)
		}
		if comment.Body != "検索対象コメント" {
			t.Errorf("comment.Body = %v, want 検索対象コメント", comment.Body)
		}
	})

	t.Run("存在しないIDの場合はnilを返す", func(t *testing.T) {
		comment, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000", spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if comment != nil {
			t.Errorf("FindByID() = %v, want nil", comment)
		}
	})

	t.Run("異なるスペースIDではnilを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("sc-find-other").
			WithName("Other Space").
			Build()

		comment, err := repo.FindByID(ctx, commentID, otherSpaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if comment != nil {
			t.Errorf("FindByID() = %v, want nil", comment)
		}
	})
}

func TestSuggestionCommentRepository_ListBySuggestionID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionCommentRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sc-list@example.com").
		WithAtname("sc_list").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sc-list-space").
		WithName("SC List Space").
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

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("コメント1").
		Build()

	time.Sleep(time.Millisecond)

	testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("コメント2").
		Build()

	t.Run("編集提案に紐づくコメント一覧を取得できる", func(t *testing.T) {
		comments, err := repo.ListBySuggestionID(ctx, suggestionID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(comments) != 2 {
			t.Fatalf("len(comments) = %v, want 2", len(comments))
		}
		if comments[0].Body != "コメント1" {
			t.Errorf("comments[0].Body = %v, want コメント1", comments[0].Body)
		}
		if comments[1].Body != "コメント2" {
			t.Errorf("comments[1].Body = %v, want コメント2", comments[1].Body)
		}
	})

	t.Run("該当なしの場合は空のスライスを返す", func(t *testing.T) {
		otherSuggestionID := testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("コメントなし提案").
			Build()

		comments, err := repo.ListBySuggestionID(ctx, otherSuggestionID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(comments) != 0 {
			t.Errorf("len(comments) = %v, want 0", len(comments))
		}
	})
}

func TestSuggestionCommentRepository_FindByNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionCommentRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sc-findnum@example.com").
		WithAtname("sc_findnum").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sc-findnum-space").
		WithName("SC FindNum Space").
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

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("番号検索対象コメント").
		Build()

	t.Run("番号でコメントを取得できる", func(t *testing.T) {
		comment, err := repo.FindByNumber(ctx, suggestionID, 1, spaceID)
		if err != nil {
			t.Fatalf("FindByNumber() error = %v", err)
		}
		if comment == nil {
			t.Fatal("FindByNumber() returned nil")
		}
		if comment.Body != "番号検索対象コメント" {
			t.Errorf("comment.Body = %v, want 番号検索対象コメント", comment.Body)
		}
		if comment.Number != 1 {
			t.Errorf("comment.Number = %v, want 1", comment.Number)
		}
	})

	t.Run("存在しない番号の場合はnilを返す", func(t *testing.T) {
		comment, err := repo.FindByNumber(ctx, suggestionID, 999, spaceID)
		if err != nil {
			t.Fatalf("FindByNumber() error = %v", err)
		}
		if comment != nil {
			t.Errorf("FindByNumber() = %v, want nil", comment)
		}
	})

	t.Run("異なるスペースIDではnilを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("sc-findnum-other").
			WithName("Other Space").
			Build()

		comment, err := repo.FindByNumber(ctx, suggestionID, 1, otherSpaceID)
		if err != nil {
			t.Fatalf("FindByNumber() error = %v", err)
		}
		if comment != nil {
			t.Errorf("FindByNumber() = %v, want nil", comment)
		}
	})
}

func TestSuggestionCommentRepository_Update(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionCommentRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sc-update@example.com").
		WithAtname("sc_update").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sc-update-space").
		WithName("SC Update Space").
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

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	commentID := testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("更新前のコメント").
		Build()

	t.Run("コメントを更新できる", func(t *testing.T) {
		comment, err := repo.Update(ctx, UpdateSuggestionCommentInput{
			ID:      commentID,
			SpaceID: spaceID,
			Body:    "更新後のコメント",
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if comment == nil {
			t.Fatal("Update() returned nil")
		}
		if comment.Body != "更新後のコメント" {
			t.Errorf("comment.Body = %v, want 更新後のコメント", comment.Body)
		}
	})

	t.Run("異なるスペースIDではnilを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("sc-update-other").
			WithName("Other Space").
			Build()

		comment, err := repo.Update(ctx, UpdateSuggestionCommentInput{
			ID:      commentID,
			SpaceID: otherSpaceID,
			Body:    "不正な更新",
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if comment != nil {
			t.Errorf("Update() = %v, want nil", comment)
		}
	})
}

func TestSuggestionCommentRepository_CountBySuggestionID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionCommentRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sc-count@example.com").
		WithAtname("sc_count").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sc-count-space").
		WithName("SC Count Space").
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

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	t.Run("コメント数を取得できる", func(t *testing.T) {
		count, err := repo.CountBySuggestionID(ctx, suggestionID, spaceID)
		if err != nil {
			t.Fatalf("CountBySuggestionID() error = %v", err)
		}
		if count != 2 {
			t.Errorf("count = %v, want 2", count)
		}
	})

	t.Run("コメントがない場合は0を返す", func(t *testing.T) {
		otherSuggestionID := testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("コメントなし提案").
			Build()

		count, err := repo.CountBySuggestionID(ctx, otherSuggestionID, spaceID)
		if err != nil {
			t.Fatalf("CountBySuggestionID() error = %v", err)
		}
		if count != 0 {
			t.Errorf("count = %v, want 0", count)
		}
	})
}
