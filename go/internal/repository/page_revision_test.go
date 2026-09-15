package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestPageRevisionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRevisionRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pagerev-create@example.com").
		WithAtname("pagerevcreate").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pagerev-create-space").
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

	t.Run("ページリビジョンを作成できる", func(t *testing.T) {
		revision, err := repo.Create(context.Background(), CreatePageRevisionInput{
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			PageID:        pageID,
			Title:         "Revision Title",
			Body:          "revision body",
			BodyHTML:      "<p>revision body</p>",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		if revision == nil {
			t.Fatal("Create()がnilを返した、期待値 = ページのリビジョン")
		}
		if revision.ID == "" {
			t.Error("revision.IDが空")
		}
		if revision.SpaceID != spaceID {
			t.Errorf("revision.SpaceID = %v、期待値 = %v", revision.SpaceID, spaceID)
		}
		if revision.SpaceMemberID != spaceMemberID {
			t.Errorf("revision.SpaceMemberID = %v、期待値 = %v", revision.SpaceMemberID, spaceMemberID)
		}
		if revision.PageID != pageID {
			t.Errorf("revision.PageID = %v、期待値 = %v", revision.PageID, pageID)
		}
		if revision.Title != "Revision Title" {
			t.Errorf("revision.Title = %v、期待値 = 'Revision Title'", revision.Title)
		}
		if revision.Body != "revision body" {
			t.Errorf("revision.Body = %v、期待値 = 'revision body'", revision.Body)
		}
		if revision.BodyHTML != "<p>revision body</p>" {
			t.Errorf("revision.BodyHTML = %v、期待値 = '<p>revision body</p>'", revision.BodyHTML)
		}
		if revision.CreatedAt.IsZero() {
			t.Error("revision.CreatedAtがゼロ値")
		}
		if revision.UpdatedAt.IsZero() {
			t.Error("revision.UpdatedAtがゼロ値")
		}
	})

	t.Run("同じページに対して複数のリビジョンを作成できる", func(t *testing.T) {
		revision1, err := repo.Create(context.Background(), CreatePageRevisionInput{
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			PageID:        pageID,
			Title:         "First Revision",
			Body:          "first body",
			BodyHTML:      "<p>first body</p>",
		})
		if err != nil {
			t.Fatalf("Create() (1件目のリビジョン) のエラー = %v", err)
		}

		revision2, err := repo.Create(context.Background(), CreatePageRevisionInput{
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			PageID:        pageID,
			Title:         "Second Revision",
			Body:          "second body",
			BodyHTML:      "<p>second body</p>",
		})
		if err != nil {
			t.Fatalf("Create() (2件目のリビジョン) のエラー = %v", err)
		}

		if revision1.ID == revision2.ID {
			t.Errorf("revision1.IDとrevision2.IDが同じ: %v", revision1.ID)
		}
		if revision1.Title != "First Revision" {
			t.Errorf("revision1.Title = %v、期待値 = 'First Revision'", revision1.Title)
		}
		if revision2.Title != "Second Revision" {
			t.Errorf("revision2.Title = %v、期待値 = 'Second Revision'", revision2.Title)
		}
	})
}
