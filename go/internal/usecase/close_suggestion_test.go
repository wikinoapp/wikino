package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCloseSuggestionUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	suggestionRepo := repository.NewSuggestionRepository(q)

	uc := NewCloseSuggestionUsecase(db, suggestionRepo)

	t.Run("正常系: オープンステータスの編集提案をクローズできる", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("close-sugg-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("close-sugg-1@example.com").
			WithAtname("closesugg1").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		suggestion := &model.Suggestion{
			ID:      suggestionID,
			SpaceID: spaceID,
			Status:  model.SuggestionStatusOpen,
		}

		output, err := uc.Execute(context.Background(), CloseSuggestionInput{
			Suggestion: suggestion,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Suggestion.Status != model.SuggestionStatusClosed {
			t.Errorf("suggestion status = %d, want %d", output.Suggestion.Status, model.SuggestionStatusClosed)
		}
	})
}
