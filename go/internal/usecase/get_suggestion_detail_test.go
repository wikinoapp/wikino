package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetSuggestionDetailUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionCommentRepo := repository.NewSuggestionCommentRepository(q)
	userRepo := repository.NewUserRepository(q)
	pageRepo := repository.NewPageRepository(q)
	uc := NewGetSuggestionDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionCommentRepo, pageRepo, userRepo)

	// テストデータのセットアップ
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-detail@example.com").
		WithAtname("sugdetailuser").
		WithName("詳細太郎").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sug-detail-space").
		WithName("SugDetail Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("詳細トピック").
		Build()

	// 編集提案を作成
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("テスト提案").
		WithBody("提案の本文").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	// ページと編集提案ページを作成
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		Build()
	testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		WithTitle("変更後タイトル").
		Build()

	// コメントを作成
	testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("テストコメント1").
		Build()

	t.Run("編集提案の詳細が取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-space",
			SuggestionNumber: 1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Suggestion == nil {
			t.Fatal("Suggestion should not be nil")
		}
		if output.Suggestion.Title != "テスト提案" {
			t.Errorf("Suggestion.Title = %q, want %q", output.Suggestion.Title, "テスト提案")
		}
		if output.Suggestion.Status != model.SuggestionStatusOpen {
			t.Errorf("Suggestion.Status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusOpen)
		}
	})

	t.Run("編集提案ページが取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-space",
			SuggestionNumber: 1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.SuggestionPages) != 1 {
			t.Fatalf("len(SuggestionPages) = %d, want 1", len(output.SuggestionPages))
		}
		if output.SuggestionPages[0].Title == nil || *output.SuggestionPages[0].Title != "変更後タイトル" {
			t.Errorf("SuggestionPages[0].Title = %v, want %q", output.SuggestionPages[0].Title, "変更後タイトル")
		}
	})

	t.Run("編集提案ページに対応する元ページが取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-space",
			SuggestionNumber: 1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.Pages) != 1 {
			t.Fatalf("len(Pages) = %d, want 1", len(output.Pages))
		}
		if output.Pages[0].ID != pageID {
			t.Errorf("Pages[0].ID = %q, want %q", output.Pages[0].ID, pageID)
		}
	})

	t.Run("コメントが取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-space",
			SuggestionNumber: 1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.Comments) != 1 {
			t.Fatalf("len(Comments) = %d, want 1", len(output.Comments))
		}
		if output.Comments[0].Body != "テストコメント1" {
			t.Errorf("Comments[0].Body = %q, want %q", output.Comments[0].Body, "テストコメント1")
		}
	})

	t.Run("作成者とコメント投稿者のユーザー情報がUserMapに含まれる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-space",
			SuggestionNumber: 1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		user, ok := output.UserMap[spaceMemberID]
		if !ok {
			t.Fatal("UserMap should contain the creator's user info")
		}
		if user.Name != "詳細太郎" {
			t.Errorf("user.Name = %q, want %q", user.Name, "詳細太郎")
		}
	})

	t.Run("ログインユーザーのSpaceMemberとTopicMemberが取得できる", func(t *testing.T) {
		// トピックメンバーを作成
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(spaceMemberID).
			Build()

		uid := userID
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-space",
			SuggestionNumber: 1,
			UserID:           &uid,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMember should not be nil for logged-in user")
		}
		if output.SpaceMember.ID != spaceMemberID {
			t.Errorf("SpaceMember.ID = %q, want %q", output.SpaceMember.ID, spaceMemberID)
		}
		if output.TopicMember == nil {
			t.Fatal("TopicMember should not be nil for topic member")
		}
	})

	t.Run("存在しないスペースの場合はnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "nonexistent",
			SuggestionNumber: 1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for nonexistent space")
		}
	})

	t.Run("存在しない編集提案番号の場合はnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-space",
			SuggestionNumber: 999,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for nonexistent suggestion number")
		}
	})
}

func TestGetSuggestionDetailUsecase_Execute_非公開トピック(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionCommentRepo := repository.NewSuggestionCommentRepository(q)
	userRepo := repository.NewUserRepository(q)
	pageRepo := repository.NewPageRepository(q)
	uc := NewGetSuggestionDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionCommentRepo, pageRepo, userRepo)

	// オーナーユーザー
	ownerUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-detail-priv-owner@example.com").
		WithAtname("sugdetailprivowner").
		Build()

	// トピックメンバーユーザー
	memberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-detail-priv-member@example.com").
		WithAtname("sugdetailprivmember").
		Build()

	// トピックメンバーでないユーザー
	nonMemberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("sug-detail-priv-nonmember@example.com").
		WithAtname("sugdetailprivnonmember").
		Build()

	// スペースとメンバーのセットアップ
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sug-detail-priv").
		WithName("SugDetail Private Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerUserID).
		Build()
	memberSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberUserID).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(nonMemberUserID).
		Build()

	// 非公開トピックを作成
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("非公開トピック").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()

	// memberUserIDをトピックメンバーに追加
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(memberSpaceMemberID).
		Build()

	// 編集提案を作成
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(memberSpaceMemberID).
		WithTitle("非公開トピックの提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	t.Run("未ログインでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-priv",
			SuggestionNumber: 1,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for unauthenticated user on private topic")
		}
	})

	t.Run("スペースオーナーは閲覧できる", func(t *testing.T) {
		uid := ownerUserID
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-priv",
			SuggestionNumber: 1,
			UserID:           &uid,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil for space owner")
		}
		if output.Suggestion.Title != "非公開トピックの提案" {
			t.Errorf("Suggestion.Title = %q, want %q", output.Suggestion.Title, "非公開トピックの提案")
		}
	})

	t.Run("トピックメンバーは閲覧できる", func(t *testing.T) {
		uid := memberUserID
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-priv",
			SuggestionNumber: 1,
			UserID:           &uid,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil for topic member")
		}
		if output.Suggestion.Title != "非公開トピックの提案" {
			t.Errorf("Suggestion.Title = %q, want %q", output.Suggestion.Title, "非公開トピックの提案")
		}
	})

	t.Run("トピックメンバーでないスペースメンバーも閲覧できる", func(t *testing.T) {
		uid := nonMemberUserID
		output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
			SpaceIdentifier:  "sug-detail-priv",
			SuggestionNumber: 1,
			UserID:           &uid,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil for space member on private topic")
		}
		if output.Suggestion.Title != "非公開トピックの提案" {
			t.Errorf("Suggestion.Title = %q, want %q", output.Suggestion.Title, "非公開トピックの提案")
		}
	})
}
