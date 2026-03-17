package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetSuggestionNewUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)

	uc := NewGetSuggestionNewUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, draftPageRepo)

	t.Run("正常系: スペースメンバーが下書きページ一覧を取得できる", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("get-sug-new-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("get-sug-new-1@example.com").
			WithAtname("getsunew1").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Get Sug New Topic 1").
			WithVisibility(0).
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page 1").
			Build()
		testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("Draft 1").
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionNewInput{
			SpaceIdentifier: "get-sug-new-1",
			TopicNumber:     1,
			UserID:          userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Space == nil {
			t.Fatal("Space should not be nil")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMember should not be nil")
		}
		if output.Topic == nil {
			t.Fatal("Topic should not be nil")
		}
		if len(output.DraftPages) != 1 {
			t.Errorf("DraftPages count = %d, want 1", len(output.DraftPages))
		}
	})

	t.Run("存在しないスペースでnilが返る", func(t *testing.T) {
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("get-sug-new-2@example.com").
			WithAtname("getsunew2").
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionNewInput{
			SpaceIdentifier: "nonexistent-space-xxx",
			TopicNumber:     1,
			UserID:          userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for nonexistent space")
		}
	})

	t.Run("スペースメンバーでない場合nilが返る", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("get-sug-new-3").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("get-sug-new-3@example.com").
			WithAtname("getsunew3").
			Build()
		testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Get Sug New Topic 3").
			WithVisibility(0).
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionNewInput{
			SpaceIdentifier: "get-sug-new-3",
			TopicNumber:     1,
			UserID:          userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-member")
		}
	})

	t.Run("非公開トピックでトピックメンバーでない場合nilが返る", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("get-sug-new-4").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("get-sug-new-4@example.com").
			WithAtname("getsunew4").
			Build()
		testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			WithRole(1). // member (not owner)
			Build()
		testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Get Sug New Topic 4").
			WithVisibility(1). // private
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionNewInput{
			SpaceIdentifier: "get-sug-new-4",
			TopicNumber:     1,
			UserID:          userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-topic-member on private topic")
		}
	})

	t.Run("非公開トピックでもスペースオーナーはアクセスできる", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("get-sug-new-5").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("get-sug-new-5@example.com").
			WithAtname("getsunew5").
			Build()
		testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			WithRole(0). // owner
			Build()
		testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Get Sug New Topic 5").
			WithVisibility(1). // private
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionNewInput{
			SpaceIdentifier: "get-sug-new-5",
			TopicNumber:     1,
			UserID:          userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil for space owner")
		}
		if output.Space == nil {
			t.Error("Space should not be nil")
		}
	})

	t.Run("編集提案にリンク済みの下書きページは除外される", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("get-sug-new-6").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("get-sug-new-6@example.com").
			WithAtname("getsunew6").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Get Sug New Topic 6").
			WithVisibility(0).
			Build()

		// 通常の下書きページ（取得されるべき）
		pageID1 := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page 1").
			Build()
		testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID1).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("Normal Draft").
			Build()

		// 編集提案にリンク済みの下書きページ（除外されるべき）
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("Linked Suggestion").
			WithStatus(model.SuggestionStatusOpen).
			Build()
		pageID2 := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Page 2").
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID2).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID2).
			WithPageRevisionID(pageRevisionID).
			Build()
		testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID2).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("Linked Draft").
			WithSuggestionPageID(suggestionPageID).
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionNewInput{
			SpaceIdentifier: "get-sug-new-6",
			TopicNumber:     1,
			UserID:          userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.DraftPages) != 1 {
			t.Errorf("DraftPages count = %d, want 1 (linked draft should be excluded)", len(output.DraftPages))
		}
	})
}
