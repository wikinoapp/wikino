package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetLatestPageRevisionsUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	uc := NewGetLatestPageRevisionsUsecase(pageRevisionRepo)

	t.Run("正常系: 各下書きページの最新リビジョンが取得できる", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("latest-rev-1").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("latest-rev-1@example.com").
			WithAtname("latestrev1").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Test Page").
			Build()

		rev := createPageRevisionForTest(t, q, spaceID, spaceMemberID, pageID)

		output, err := uc.Execute(context.Background(), GetLatestPageRevisionsInput{
			DraftPages: []*model.DraftPage{
				{PageID: pageID},
			},
			SpaceID: spaceID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.PageRevisions) != 1 {
			t.Fatalf("PageRevisions count = %d, want 1", len(output.PageRevisions))
		}
		gotRevision := output.PageRevisions[pageID]
		if gotRevision == nil {
			t.Fatal("PageRevision for pageID should not be nil")
		}
		if gotRevision.ID != rev.ID {
			t.Errorf("PageRevision.ID = %v, want %v", gotRevision.ID, rev.ID)
		}
	})

	t.Run("正常系: 複数ページのリビジョンが取得できる", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("latest-rev-2").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("latest-rev-2@example.com").
			WithAtname("latestrev2").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		page1ID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page 1").
			Build()
		page2ID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Page 2").
			Build()

		createPageRevisionForTest(t, q, spaceID, spaceMemberID, page1ID)
		createPageRevisionForTest(t, q, spaceID, spaceMemberID, page2ID)

		output, err := uc.Execute(context.Background(), GetLatestPageRevisionsInput{
			DraftPages: []*model.DraftPage{
				{PageID: page1ID},
				{PageID: page2ID},
			},
			SpaceID: spaceID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.PageRevisions) != 2 {
			t.Fatalf("PageRevisions count = %d, want 2", len(output.PageRevisions))
		}
		if output.PageRevisions[page1ID] == nil {
			t.Error("PageRevision for page1ID should not be nil")
		}
		if output.PageRevisions[page2ID] == nil {
			t.Error("PageRevision for page2ID should not be nil")
		}
	})

	t.Run("異常系: ページリビジョンが存在しない場合はエラー", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("latest-rev-err").
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("No Revision Page").
			WithUnpublished().
			Build()

		_, err := uc.Execute(context.Background(), GetLatestPageRevisionsInput{
			DraftPages: []*model.DraftPage{
				{PageID: pageID},
			},
			SpaceID: spaceID,
		})
		if err == nil {
			t.Error("expected error but got nil")
		}
	})
}
