package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestPageRepository_FindBySpaceAndNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-find-space").
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
		WithBody("Hello").
		WithBodyHTML("<p>Hello</p>").
		Build()

	t.Run("存在するページをスペースIDとページ番号で取得できる", func(t *testing.T) {
		page, err := repo.FindBySpaceAndNumber(context.Background(), spaceID, 1)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if page == nil {
			t.Fatal("FindBySpaceAndNumber() returned nil, want page")
		}
		if page.ID != pageID {
			t.Errorf("page.ID = %v, want %v", page.ID, pageID)
		}
		if page.SpaceID != spaceID {
			t.Errorf("page.SpaceID = %v, want %v", page.SpaceID, spaceID)
		}
		if page.TopicID != topicID {
			t.Errorf("page.TopicID = %v, want %v", page.TopicID, topicID)
		}
		if page.Number != 1 {
			t.Errorf("page.Number = %v, want 1", page.Number)
		}
		if page.Title == nil || *page.Title != "Test Page" {
			t.Errorf("page.Title = %v, want 'Test Page'", page.Title)
		}
		if page.Body != "Hello" {
			t.Errorf("page.Body = %v, want 'Hello'", page.Body)
		}
		if page.BodyHTML != "<p>Hello</p>" {
			t.Errorf("page.BodyHTML = %v, want '<p>Hello</p>'", page.BodyHTML)
		}
		if page.PublishedAt == nil {
			t.Error("page.PublishedAt should not be nil")
		}
		if page.DiscardedAt != nil {
			t.Errorf("page.DiscardedAt = %v, want nil", page.DiscardedAt)
		}
	})

	t.Run("存在しないページ番号はnilを返す", func(t *testing.T) {
		page, err := repo.FindBySpaceAndNumber(context.Background(), spaceID, 999)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if page != nil {
			t.Errorf("FindBySpaceAndNumber() = %v, want nil", page)
		}
	})

	t.Run("廃棄されたページは取得できない", func(t *testing.T) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(99).
			WithTitle("Discarded Page").
			WithDiscarded().
			Build()

		page, err := repo.FindBySpaceAndNumber(context.Background(), spaceID, 99)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if page != nil {
			t.Errorf("FindBySpaceAndNumber() = %v, want nil (discarded page should not be returned)", page)
		}
	})
}

func TestPageRepository_FindByIDs(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-find-ids-space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	pageID1 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Page 1").
		Build()

	pageID2 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Page 2").
		Build()

	// 非公開ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Unpublished").
		WithUnpublished().
		Build()

	t.Run("IDリストに含まれる公開済みページを取得できる", func(t *testing.T) {
		pages, err := repo.FindByIDs(context.Background(), []model.PageID{pageID1, pageID2}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(pages) != 2 {
			t.Fatalf("len(pages) = %v, want 2", len(pages))
		}
		if pages[0].Number != 1 {
			t.Errorf("pages[0].Number = %v, want 1", pages[0].Number)
		}
		if pages[1].Number != 2 {
			t.Errorf("pages[1].Number = %v, want 2", pages[1].Number)
		}
	})

	t.Run("非公開ページは結果に含まれない", func(t *testing.T) {
		unpublishedID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(4).
			WithTitle("Also Unpublished").
			WithUnpublished().
			Build()

		pages, err := repo.FindByIDs(context.Background(), []model.PageID{unpublishedID}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(pages) != 0 {
			t.Errorf("len(pages) = %v, want 0", len(pages))
		}
	})

	t.Run("空のIDリストは空のスライスを返す", func(t *testing.T) {
		pages, err := repo.FindByIDs(context.Background(), []model.PageID{}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDs() error = %v", err)
		}
		if len(pages) != 0 {
			t.Errorf("len(pages) = %v, want 0", len(pages))
		}
	})
}

func TestPageRepository_FindBacklinkedByPageID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-backlink-space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	// 被リンクページ
	targetPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Target Page").
		Build()

	// targetPageIDをリンクしているページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linking Page").
		WithLinkedPageIDs([]model.PageID{targetPageID}).
		Build()

	// targetPageIDをリンクしていないページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("No Link Page").
		Build()

	t.Run("バックリンクページを取得できる", func(t *testing.T) {
		pages, err := repo.FindBacklinkedByPageID(context.Background(), targetPageID, spaceID)
		if err != nil {
			t.Fatalf("FindBacklinkedByPageID() error = %v", err)
		}
		if len(pages) != 1 {
			t.Fatalf("len(pages) = %v, want 1", len(pages))
		}
		if *pages[0].Title != "Linking Page" {
			t.Errorf("pages[0].Title = %v, want 'Linking Page'", *pages[0].Title)
		}
	})

	t.Run("バックリンクがないページは空のスライスを返す", func(t *testing.T) {
		noLinkPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(4).
			WithTitle("Isolated Page").
			Build()

		pages, err := repo.FindBacklinkedByPageID(context.Background(), noLinkPageID, spaceID)
		if err != nil {
			t.Fatalf("FindBacklinkedByPageID() error = %v", err)
		}
		if len(pages) != 0 {
			t.Errorf("len(pages) = %v, want 0", len(pages))
		}
	})
}

func TestPageRepository_FindByTopicAndTitle(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-topic-title-space").
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
		WithTitle("My Page").
		Build()

	t.Run("トピックIDとタイトルでページを取得できる", func(t *testing.T) {
		page, err := repo.FindByTopicAndTitle(context.Background(), topicID, "My Page", spaceID)
		if err != nil {
			t.Fatalf("FindByTopicAndTitle() error = %v", err)
		}
		if page == nil {
			t.Fatal("FindByTopicAndTitle() returned nil, want page")
		}
		if page.ID != pageID {
			t.Errorf("page.ID = %v, want %v", page.ID, pageID)
		}
	})

	t.Run("存在しないタイトルはnilを返す", func(t *testing.T) {
		page, err := repo.FindByTopicAndTitle(context.Background(), topicID, "Not Exist", spaceID)
		if err != nil {
			t.Fatalf("FindByTopicAndTitle() error = %v", err)
		}
		if page != nil {
			t.Errorf("FindByTopicAndTitle() = %v, want nil", page)
		}
	})
}

func TestPageRepository_Update(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-update-space").
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
		WithTitle("Before Update").
		WithBody("old body").
		WithBodyHTML("<p>old body</p>").
		Build()

	t.Run("ページを更新できる", func(t *testing.T) {
		now := time.Now()
		newTitle := "After Update"
		page, err := repo.Update(context.Background(), UpdatePageInput{
			ID:            pageID,
			SpaceID:       spaceID,
			TopicID:       topicID,
			Title:         &newTitle,
			Body:          "new body",
			BodyHTML:      "<p>new body</p>",
			LinkedPageIDs: []model.PageID{},
			ModifiedAt:    now,
			PublishedAt:   &now,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if page == nil {
			t.Fatal("Update() returned nil, want page")
		}
		if page.Title == nil || *page.Title != "After Update" {
			t.Errorf("page.Title = %v, want 'After Update'", page.Title)
		}
		if page.Body != "new body" {
			t.Errorf("page.Body = %v, want 'new body'", page.Body)
		}
		if page.BodyHTML != "<p>new body</p>" {
			t.Errorf("page.BodyHTML = %v, want '<p>new body</p>'", page.BodyHTML)
		}
	})
}

func TestPageRepository_CreateLinkedPage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewPageRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-create-linked-space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	t.Run("Wikiリンクからページを作成できる", func(t *testing.T) {
		page, err := repo.CreateLinkedPage(context.Background(), CreateLinkedPageInput{
			SpaceID: spaceID,
			TopicID: topicID,
			Number:  100,
			Title:   "Linked Page",
		})
		if err != nil {
			t.Fatalf("CreateLinkedPage() error = %v", err)
		}
		if page == nil {
			t.Fatal("CreateLinkedPage() returned nil, want page")
		}
		if page.SpaceID != spaceID {
			t.Errorf("page.SpaceID = %v, want %v", page.SpaceID, spaceID)
		}
		if page.TopicID != topicID {
			t.Errorf("page.TopicID = %v, want %v", page.TopicID, topicID)
		}
		if page.Number != 100 {
			t.Errorf("page.Number = %v, want 100", page.Number)
		}
		if page.Title == nil || *page.Title != "Linked Page" {
			t.Errorf("page.Title = %v, want 'Linked Page'", page.Title)
		}
		if page.Body != "" {
			t.Errorf("page.Body = %v, want empty string", page.Body)
		}
		if page.BodyHTML != "" {
			t.Errorf("page.BodyHTML = %v, want empty string", page.BodyHTML)
		}
		if page.PublishedAt == nil {
			t.Error("page.PublishedAt should not be nil")
		}
	})
}
