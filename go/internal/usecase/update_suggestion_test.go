package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdateSuggestionUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	suggestionRepo := repository.NewSuggestionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	pageRepo := repository.NewPageRepository(q)

	uc := NewUpdateSuggestionUsecase(db, suggestionRepo, topicRepo, pageRepo)

	t.Run("正常系: タイトルと本文を更新できる", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sug-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sug-1@example.com").
			WithAtname("updatesug1").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("旧タイトル").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		output, err := uc.Execute(ctx, UpdateSuggestionInput{
			SuggestionID:     suggestionID,
			SpaceID:          spaceID,
			SpaceIdentifier:  "update-sug-1",
			CurrentTopicName: "テストトピック",
			Title:            "新タイトル",
			Body:             "新本文",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.Suggestion.Title != "新タイトル" {
			t.Errorf("Title = %q, want %q", output.Suggestion.Title, "新タイトル")
		}
		if output.Suggestion.Body != "新本文" {
			t.Errorf("Body = %q, want %q", output.Suggestion.Body, "新本文")
		}
		if output.Suggestion.BodyHTML == "" {
			t.Error("BodyHTML should not be empty")
		}
	})

	t.Run("正常系: 本文を空に更新できる", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sug-2").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sug-2@example.com").
			WithAtname("updatesug2").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック2").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		output, err := uc.Execute(ctx, UpdateSuggestionInput{
			SuggestionID:     suggestionID,
			SpaceID:          spaceID,
			SpaceIdentifier:  "update-sug-2",
			CurrentTopicName: "テストトピック2",
			Title:            "更新タイトル",
			Body:             "",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.Suggestion.Title != "更新タイトル" {
			t.Errorf("Title = %q, want %q", output.Suggestion.Title, "更新タイトル")
		}
		if output.Suggestion.Body != "" {
			t.Errorf("Body = %q, want empty", output.Suggestion.Body)
		}
	})

	t.Run("正常系: Markdown本文がHTMLに変換される", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("update-sug-3").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("update-sug-3@example.com").
			WithAtname("updatesug3").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("テストトピック3").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("テスト").
			WithStatus(model.SuggestionStatusOpen).
			Build()

		output, err := uc.Execute(ctx, UpdateSuggestionInput{
			SuggestionID:     suggestionID,
			SpaceID:          spaceID,
			SpaceIdentifier:  "update-sug-3",
			CurrentTopicName: "テストトピック3",
			Title:            "Markdownテスト",
			Body:             "**太字** テスト",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.Suggestion.BodyHTML == "" {
			t.Error("BodyHTML should not be empty")
		}
		if !strings.Contains(output.Suggestion.BodyHTML, "<strong>") {
			t.Errorf("BodyHTML should contain <strong> tag, got: %s", output.Suggestion.BodyHTML)
		}
	})
}
