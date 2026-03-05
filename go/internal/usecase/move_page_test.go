package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestMovePageUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	uc := NewMovePageUsecase(db, pageRepo)

	t.Run("正常系: ページを別のトピックに移動できる", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("move-test-1").
			Build()
		srcTopicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Source Topic").
			WithNumber(1).
			Build()
		destTopicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Dest Topic").
			WithNumber(2).
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(srcTopicID).
			WithNumber(1).
			WithTitle("Move Test Page").
			Build()

		output, err := uc.Execute(context.Background(), MovePageInput{
			PageID:      pageID,
			SpaceID:     spaceID,
			DestTopicID: destTopicID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.Page == nil {
			t.Fatal("output.Page should not be nil")
		}
		if output.Page.TopicID != destTopicID {
			t.Errorf("Page.TopicID = %v, want %v", output.Page.TopicID, destTopicID)
		}
		if output.Page.SpaceID != spaceID {
			t.Errorf("Page.SpaceID = %v, want %v", output.Page.SpaceID, spaceID)
		}
	})

	t.Run("異常系: 存在しないページIDでエラーになる", func(t *testing.T) {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("move-test-2").
			Build()
		destTopicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Dest Topic").
			Build()

		_, err := uc.Execute(context.Background(), MovePageInput{
			PageID:      "nonexistent-page-id",
			SpaceID:     spaceID,
			DestTopicID: destTopicID,
		})
		if err == nil {
			t.Fatal("Execute() error = nil, want error")
		}
	})
}
