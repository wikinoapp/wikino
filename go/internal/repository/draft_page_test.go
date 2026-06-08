package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestDraftPageRepository_FindByPageAndMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-find@example.com").
		WithAtname("draftfind").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-find-space").
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

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		WithBodyHTML("<p>Draft body</p>").
		Build()

	t.Run("ページIDとスペースメンバーIDで下書きを取得できる", func(t *testing.T) {
		draft, err := repo.FindByPageAndMember(context.Background(), pageID, spaceMemberID, spaceID)
		if err != nil {
			t.Fatalf("FindByPageAndMember() error = %v", err)
		}
		if draft == nil {
			t.Fatal("FindByPageAndMember() returned nil, want draft page")
		}
		if draft.ID != draftPageID {
			t.Errorf("draft.ID = %v, want %v", draft.ID, draftPageID)
		}
		if draft.SpaceID != spaceID {
			t.Errorf("draft.SpaceID = %v, want %v", draft.SpaceID, spaceID)
		}
		if draft.PageID != pageID {
			t.Errorf("draft.PageID = %v, want %v", draft.PageID, pageID)
		}
		if draft.SpaceMemberID != spaceMemberID {
			t.Errorf("draft.SpaceMemberID = %v, want %v", draft.SpaceMemberID, spaceMemberID)
		}
		if draft.TopicID != topicID {
			t.Errorf("draft.TopicID = %v, want %v", draft.TopicID, topicID)
		}
		if draft.Title == nil || *draft.Title != "Draft Title" {
			t.Errorf("draft.Title = %v, want 'Draft Title'", draft.Title)
		}
		if draft.Body != "Draft body" {
			t.Errorf("draft.Body = %v, want 'Draft body'", draft.Body)
		}
		if draft.BodyHTML != "<p>Draft body</p>" {
			t.Errorf("draft.BodyHTML = %v, want '<p>Draft body</p>'", draft.BodyHTML)
		}
	})

	t.Run("存在しないページIDはnilを返す", func(t *testing.T) {
		draft, err := repo.FindByPageAndMember(context.Background(), "00000000-0000-0000-0000-000000000000", spaceMemberID, spaceID)
		if err != nil {
			t.Fatalf("FindByPageAndMember() error = %v", err)
		}
		if draft != nil {
			t.Errorf("FindByPageAndMember() = %v, want nil", draft)
		}
	})

	t.Run("存在しないスペースメンバーIDはnilを返す", func(t *testing.T) {
		draft, err := repo.FindByPageAndMember(context.Background(), pageID, "00000000-0000-0000-0000-000000000000", spaceID)
		if err != nil {
			t.Fatalf("FindByPageAndMember() error = %v", err)
		}
		if draft != nil {
			t.Errorf("FindByPageAndMember() = %v, want nil", draft)
		}
	})
}

func TestDraftPageRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-create@example.com").
		WithAtname("draftcreate").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-create-space").
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

	t.Run("下書きを作成できる", func(t *testing.T) {
		now := time.Now()
		title := "New Draft"
		draft, err := repo.Create(context.Background(), CreateDraftPageInput{
			SpaceID:       spaceID,
			PageID:        pageID,
			SpaceMemberID: spaceMemberID,
			TopicID:       topicID,
			Title:         &title,
			Body:          "draft body",
			BodyHTML:      "<p>draft body</p>",
			LinkedPageIDs: []model.PageID{},
			ModifiedAt:    now,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if draft == nil {
			t.Fatal("Create() returned nil, want draft page")
		}
		if draft.ID == "" {
			t.Error("draft.ID should not be empty")
		}
		if draft.SpaceID != spaceID {
			t.Errorf("draft.SpaceID = %v, want %v", draft.SpaceID, spaceID)
		}
		if draft.PageID != pageID {
			t.Errorf("draft.PageID = %v, want %v", draft.PageID, pageID)
		}
		if draft.SpaceMemberID != spaceMemberID {
			t.Errorf("draft.SpaceMemberID = %v, want %v", draft.SpaceMemberID, spaceMemberID)
		}
		if draft.Title == nil || *draft.Title != "New Draft" {
			t.Errorf("draft.Title = %v, want 'New Draft'", draft.Title)
		}
		if draft.Body != "draft body" {
			t.Errorf("draft.Body = %v, want 'draft body'", draft.Body)
		}
		if draft.BodyHTML != "<p>draft body</p>" {
			t.Errorf("draft.BodyHTML = %v, want '<p>draft body</p>'", draft.BodyHTML)
		}
	})

	t.Run("タイトルがnilの下書きを作成できる", func(t *testing.T) {
		pageID2 := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Page 2").
			Build()

		now := time.Now()
		draft, err := repo.Create(context.Background(), CreateDraftPageInput{
			SpaceID:       spaceID,
			PageID:        pageID2,
			SpaceMemberID: spaceMemberID,
			TopicID:       topicID,
			Title:         nil,
			Body:          "no title draft",
			BodyHTML:      "<p>no title draft</p>",
			LinkedPageIDs: []model.PageID{},
			ModifiedAt:    now,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if draft.Title != nil {
			t.Errorf("draft.Title = %v, want nil", draft.Title)
		}
	})
}

func TestDraftPageRepository_Update(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-update@example.com").
		WithAtname("draftupdate").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-update-space").
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

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Before Update").
		WithBody("old body").
		WithBodyHTML("<p>old body</p>").
		Build()

	t.Run("下書きを更新できる", func(t *testing.T) {
		now := time.Now()
		newTitle := "After Update"
		draft, err := repo.Update(context.Background(), UpdateDraftPageInput{
			ID:            draftPageID,
			SpaceID:       spaceID,
			TopicID:       topicID,
			Title:         &newTitle,
			Body:          "new body",
			BodyHTML:      "<p>new body</p>",
			LinkedPageIDs: []model.PageID{},
			ModifiedAt:    now,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if draft == nil {
			t.Fatal("Update() returned nil, want draft page")
		}
		if draft.Title == nil || *draft.Title != "After Update" {
			t.Errorf("draft.Title = %v, want 'After Update'", draft.Title)
		}
		if draft.Body != "new body" {
			t.Errorf("draft.Body = %v, want 'new body'", draft.Body)
		}
		if draft.BodyHTML != "<p>new body</p>" {
			t.Errorf("draft.BodyHTML = %v, want '<p>new body</p>'", draft.BodyHTML)
		}
	})
}

func TestDraftPageRepository_Delete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-delete@example.com").
		WithAtname("draftdelete").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-delete-space").
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

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		Build()

	t.Run("下書きを削除できる", func(t *testing.T) {
		err := repo.Delete(context.Background(), draftPageID, spaceID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// 削除後に取得できないことを確認
		draft, err := repo.FindByPageAndMember(context.Background(), pageID, spaceMemberID, spaceID)
		if err != nil {
			t.Fatalf("FindByPageAndMember() error = %v", err)
		}
		if draft != nil {
			t.Errorf("FindByPageAndMember() = %v, want nil (deleted draft should not be returned)", draft)
		}
	})
}

func TestDraftPageRepository_CreateWithSuggestionPageID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-create-sp@example.com").
		WithAtname("draftcreatesp").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-create-sp").
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

	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTitle("Test Page").
		WithBody("body").
		WithBodyHTML("<p>body</p>").
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

	t.Run("suggestion_page_idを指定して下書きを作成できる", func(t *testing.T) {
		now := time.Now()
		title := "Draft with suggestion"
		draft, err := repo.Create(context.Background(), CreateDraftPageInput{
			SpaceID:          spaceID,
			PageID:           pageID,
			SpaceMemberID:    spaceMemberID,
			TopicID:          topicID,
			SuggestionPageID: &suggestionPageID,
			Title:            &title,
			Body:             "body",
			BodyHTML:         "<p>body</p>",
			LinkedPageIDs:    []model.PageID{},
			ModifiedAt:       now,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if draft.SuggestionPageID == nil {
			t.Fatal("draft.SuggestionPageID should not be nil")
		}
		if *draft.SuggestionPageID != suggestionPageID {
			t.Errorf("draft.SuggestionPageID = %v, want %v", *draft.SuggestionPageID, suggestionPageID)
		}
	})
}

func TestDraftPageRepository_UpdateSuggestionPageID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-update-sp@example.com").
		WithAtname("draftupdatesp").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-update-sp").
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

	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTitle("Test Page").
		WithBody("body").
		WithBodyHTML("<p>body</p>").
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

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		Build()

	t.Run("suggestion_page_idを設定できる", func(t *testing.T) {
		draft, err := repo.UpdateSuggestionPageID(context.Background(), draftPageID, spaceID, &suggestionPageID)
		if err != nil {
			t.Fatalf("UpdateSuggestionPageID() error = %v", err)
		}
		if draft.SuggestionPageID == nil {
			t.Fatal("draft.SuggestionPageID should not be nil")
		}
		if *draft.SuggestionPageID != suggestionPageID {
			t.Errorf("draft.SuggestionPageID = %v, want %v", *draft.SuggestionPageID, suggestionPageID)
		}
	})

	t.Run("suggestion_page_idをクリアできる", func(t *testing.T) {
		draft, err := repo.UpdateSuggestionPageID(context.Background(), draftPageID, spaceID, nil)
		if err != nil {
			t.Fatalf("UpdateSuggestionPageID() error = %v", err)
		}
		if draft.SuggestionPageID != nil {
			t.Errorf("draft.SuggestionPageID = %v, want nil", *draft.SuggestionPageID)
		}
	})
}

func TestDraftPageRepository_CreateWithFeaturedImageAttachmentID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-create-fimg@example.com").
		WithAtname("draftcreatefimg").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-create-fimg").
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

	attachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	t.Run("featured_image_attachment_idを指定して下書きを作成できる", func(t *testing.T) {
		now := time.Now()
		title := "Draft with featured image"
		draft, err := repo.Create(context.Background(), CreateDraftPageInput{
			SpaceID:                   spaceID,
			PageID:                    pageID,
			SpaceMemberID:             spaceMemberID,
			TopicID:                   topicID,
			Title:                     &title,
			Body:                      "body",
			BodyHTML:                  "<p>body</p>",
			LinkedPageIDs:             []model.PageID{},
			FeaturedImageAttachmentID: &attachmentID,
			ModifiedAt:                now,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if draft.FeaturedImageAttachmentID == nil {
			t.Fatal("draft.FeaturedImageAttachmentID should not be nil")
		}
		if *draft.FeaturedImageAttachmentID != attachmentID {
			t.Errorf("draft.FeaturedImageAttachmentID = %v, want %v", *draft.FeaturedImageAttachmentID, attachmentID)
		}
	})

	t.Run("featured_image_attachment_idがnilの下書きを作成できる", func(t *testing.T) {
		pageID2 := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Page 2").
			Build()

		now := time.Now()
		title := "Draft without featured image"
		draft, err := repo.Create(context.Background(), CreateDraftPageInput{
			SpaceID:                   spaceID,
			PageID:                    pageID2,
			SpaceMemberID:             spaceMemberID,
			TopicID:                   topicID,
			Title:                     &title,
			Body:                      "body",
			BodyHTML:                  "<p>body</p>",
			LinkedPageIDs:             []model.PageID{},
			FeaturedImageAttachmentID: nil,
			ModifiedAt:                now,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if draft.FeaturedImageAttachmentID != nil {
			t.Errorf("draft.FeaturedImageAttachmentID = %v, want nil", *draft.FeaturedImageAttachmentID)
		}
	})
}

func TestDraftPageRepository_UpdateWithFeaturedImageAttachmentID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-update-fimg@example.com").
		WithAtname("draftupdatefimg").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-update-fimg").
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

	attachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Before Update").
		WithBody("old body").
		WithBodyHTML("<p>old body</p>").
		Build()

	t.Run("featured_image_attachment_idを設定して下書きを更新できる", func(t *testing.T) {
		now := time.Now()
		newTitle := "After Update"
		draft, err := repo.Update(context.Background(), UpdateDraftPageInput{
			ID:                        draftPageID,
			SpaceID:                   spaceID,
			TopicID:                   topicID,
			Title:                     &newTitle,
			Body:                      "new body",
			BodyHTML:                  "<p>new body</p>",
			LinkedPageIDs:             []model.PageID{},
			FeaturedImageAttachmentID: &attachmentID,
			ModifiedAt:                now,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if draft.FeaturedImageAttachmentID == nil {
			t.Fatal("draft.FeaturedImageAttachmentID should not be nil")
		}
		if *draft.FeaturedImageAttachmentID != attachmentID {
			t.Errorf("draft.FeaturedImageAttachmentID = %v, want %v", *draft.FeaturedImageAttachmentID, attachmentID)
		}
	})

	t.Run("featured_image_attachment_idをnilにクリアできる", func(t *testing.T) {
		now := time.Now()
		title := "Cleared"
		draft, err := repo.Update(context.Background(), UpdateDraftPageInput{
			ID:                        draftPageID,
			SpaceID:                   spaceID,
			TopicID:                   topicID,
			Title:                     &title,
			Body:                      "body",
			BodyHTML:                  "<p>body</p>",
			LinkedPageIDs:             []model.PageID{},
			FeaturedImageAttachmentID: nil,
			ModifiedAt:                now,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if draft.FeaturedImageAttachmentID != nil {
			t.Errorf("draft.FeaturedImageAttachmentID = %v, want nil", *draft.FeaturedImageAttachmentID)
		}
	})
}

func TestDraftPageRepository_ListByUserForIndex(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-index@example.com").
		WithAtname("draftindex").
		Build()

	// スペースBを先に作成（名前順ソートでBが後に来ることを確認するため）
	spaceB := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-idx-b").
		WithName("Space B").
		Build()
	memberB := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceB).
		WithUserID(userID).
		Build()
	topicB1 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceB).
		WithNumber(1).
		WithName("Topic B1").
		Build()
	pageBID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceB).
		WithTopicID(topicB1).
		WithNumber(1).
		WithTitle("Page B").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceB).
		WithPageID(pageBID).
		WithSpaceMemberID(memberB).
		WithTopicID(topicB1).
		WithTitle("Draft B").
		Build()

	// スペースA（名前順ソートでAが先に来る）
	spaceA := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-idx-a").
		WithName("Space A").
		Build()
	memberA := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceA).
		WithUserID(userID).
		Build()
	topicA1 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceA).
		WithNumber(1).
		WithName("Topic A1").
		Build()
	pageAID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceA).
		WithTopicID(topicA1).
		WithNumber(1).
		WithTitle("Page A").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceA).
		WithPageID(pageAID).
		WithSpaceMemberID(memberA).
		WithTopicID(topicA1).
		WithTitle("Draft A").
		Build()

	t.Run("スペース名・トピック名の順にソートされた下書き一覧を取得できる", func(t *testing.T) {
		drafts, err := repo.ListByUserForIndex(context.Background(), userID)
		if err != nil {
			t.Fatalf("ListByUserForIndex() error = %v", err)
		}
		if len(drafts) != 2 {
			t.Fatalf("ListByUserForIndex() returned %d drafts, want 2", len(drafts))
		}

		// Space A が先、Space B が後
		if drafts[0].Title == nil || *drafts[0].Title != "Draft A" {
			t.Errorf("drafts[0].Title = %v, want 'Draft A'", drafts[0].Title)
		}
		if drafts[1].Title == nil || *drafts[1].Title != "Draft B" {
			t.Errorf("drafts[1].Title = %v, want 'Draft B'", drafts[1].Title)
		}

		// スペース名・トピック名・IDが設定されていることを確認
		if drafts[0].Topic == nil || drafts[0].Topic.Space == nil {
			t.Fatal("drafts[0].Topic.Space should not be nil")
		}
		if drafts[0].Topic.Space.Name != "Space A" {
			t.Errorf("drafts[0].Topic.Space.Name = %v, want 'Space A'", drafts[0].Topic.Space.Name)
		}
		if string(drafts[0].Topic.Space.ID) == "" {
			t.Error("drafts[0].Topic.Space.ID should not be empty")
		}
		if drafts[0].Topic.Name != "Topic A1" {
			t.Errorf("drafts[0].Topic.Name = %v, want 'Topic A1'", drafts[0].Topic.Name)
		}
		if string(drafts[0].Topic.ID) == "" {
			t.Error("drafts[0].Topic.ID should not be empty")
		}
	})

	t.Run("下書きがないユーザーは空スライスを返す", func(t *testing.T) {
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("draft-index-none@example.com").
			WithAtname("draftindexnone").
			Build()

		drafts, err := repo.ListByUserForIndex(context.Background(), otherUserID)
		if err != nil {
			t.Fatalf("ListByUserForIndex() error = %v", err)
		}
		if len(drafts) != 0 {
			t.Errorf("ListByUserForIndex() returned %d drafts, want 0", len(drafts))
		}
	})
}

func TestDraftPageRepository_ListByUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-list@example.com").
		WithAtname("draftlist").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-list-space").
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

	// 下書きを3件作成（modified_atの順序をテストするため時間をずらす）
	now := time.Now()

	pageID1 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Page 1").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID1).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft 1").
		WithModifiedAt(now.Add(-2 * time.Hour)).
		Build()

	pageID2 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Page 2").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID2).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft 2").
		WithModifiedAt(now).
		Build()

	pageID3 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Page 3").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID3).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft 3").
		WithModifiedAt(now.Add(-1 * time.Hour)).
		Build()

	t.Run("ユーザーの下書き一覧をmodified_at降順で取得できる", func(t *testing.T) {
		drafts, err := repo.ListByUser(context.Background(), userID, 5)
		if err != nil {
			t.Fatalf("ListByUser() error = %v", err)
		}
		if len(drafts) != 3 {
			t.Fatalf("ListByUser() returned %d drafts, want 3", len(drafts))
		}

		// modified_at DESC でソートされていることを確認
		if drafts[0].Title == nil || *drafts[0].Title != "Draft 2" {
			t.Errorf("drafts[0].Title = %v, want 'Draft 2'", drafts[0].Title)
		}
		if drafts[1].Title == nil || *drafts[1].Title != "Draft 3" {
			t.Errorf("drafts[1].Title = %v, want 'Draft 3'", drafts[1].Title)
		}
		if drafts[2].Title == nil || *drafts[2].Title != "Draft 1" {
			t.Errorf("drafts[2].Title = %v, want 'Draft 1'", drafts[2].Title)
		}

		// 関連エンティティの情報が正しいことを確認
		if drafts[0].Page == nil {
			t.Fatal("drafts[0].Page should not be nil")
		}
		if drafts[0].Page.Number != 2 {
			t.Errorf("drafts[0].Page.Number = %v, want 2", drafts[0].Page.Number)
		}
		if drafts[0].Topic == nil {
			t.Fatal("drafts[0].Topic should not be nil")
		}
		if drafts[0].Topic.Name != "General" {
			t.Errorf("drafts[0].Topic.Name = %v, want 'General'", drafts[0].Topic.Name)
		}
		if drafts[0].Topic.Space == nil {
			t.Fatal("drafts[0].Topic.Space should not be nil")
		}
		if string(drafts[0].Topic.Space.Identifier) != "draft-list-space" {
			t.Errorf("drafts[0].Topic.Space.Identifier = %v, want 'draft-list-space'", drafts[0].Topic.Space.Identifier)
		}
		// Verify Space.Name is populated for the home page CardLinkDraftPage. Builder default is "Test Space".
		// [Ja] ホーム画面のカードでスペース名を表示するため Space.Name が設定されていることを確認する。SpaceBuilder のデフォルト名は "Test Space"。
		if drafts[0].Topic.Space.Name != "Test Space" {
			t.Errorf("drafts[0].Topic.Space.Name = %v, want 'Test Space'", drafts[0].Topic.Space.Name)
		}
	})

	t.Run("LIMITが適用される", func(t *testing.T) {
		drafts, err := repo.ListByUser(context.Background(), userID, 2)
		if err != nil {
			t.Fatalf("ListByUser() error = %v", err)
		}
		if len(drafts) != 2 {
			t.Fatalf("ListByUser() returned %d drafts, want 2", len(drafts))
		}
	})

	t.Run("下書きがないユーザーは空スライスを返す", func(t *testing.T) {
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("draft-list-nodata@example.com").
			WithAtname("draftlistnodata").
			Build()

		drafts, err := repo.ListByUser(context.Background(), otherUserID, 5)
		if err != nil {
			t.Fatalf("ListByUser() error = %v", err)
		}
		if len(drafts) != 0 {
			t.Errorf("ListByUser() returned %d drafts, want 0", len(drafts))
		}
	})

	t.Run("論理削除されたページの下書きは除外される", func(t *testing.T) {
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("draft-list-discard@example.com").
			WithAtname("draftlistdiscard").
			Build()

		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("draft-list-discard").
			Build()

		otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(otherSpaceID).
			WithUserID(otherUserID).
			Build()

		otherTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(otherSpaceID).
			WithNumber(1).
			WithName("General").
			Build()

		discardedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(otherSpaceID).
			WithTopicID(otherTopicID).
			WithNumber(1).
			WithTitle("Discarded Page").
			WithDiscarded().
			Build()

		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(otherSpaceID).
			WithPageID(discardedPageID).
			WithSpaceMemberID(otherMemberID).
			WithTopicID(otherTopicID).
			WithTitle("Draft on discarded page").
			Build()

		drafts, err := repo.ListByUser(context.Background(), otherUserID, 5)
		if err != nil {
			t.Fatalf("ListByUser() error = %v", err)
		}
		if len(drafts) != 0 {
			t.Errorf("ListByUser() returned %d drafts, want 0 (discarded page should be excluded)", len(drafts))
		}
	})
}

func TestDraftPageRepository_ListBySpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-sm-list@example.com").
		WithAtname("draftsmlist").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-sm-space").
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

	now := time.Now()

	// Create 3 drafts with staggered modified_at to test ordering.
	// [Ja] 下書きを 3 件作成 (modified_at の順序をテストするため時間をずらす)。
	pageID1 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Page 1").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID1).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft 1").
		WithModifiedAt(now.Add(-2 * time.Hour)).
		Build()

	pageID2 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Page 2").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID2).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft 2").
		WithModifiedAt(now).
		Build()

	pageID3 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Page 3").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID3).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft 3").
		WithModifiedAt(now.Add(-1 * time.Hour)).
		Build()

	t.Run("スペースメンバーの下書き一覧をmodified_at降順で取得できる", func(t *testing.T) {
		drafts, err := repo.ListBySpaceMember(context.Background(), spaceMemberID, spaceID, 20)
		if err != nil {
			t.Fatalf("ListBySpaceMember() error = %v", err)
		}
		if len(drafts) != 3 {
			t.Fatalf("ListBySpaceMember() returned %d drafts, want 3", len(drafts))
		}

		if drafts[0].Title == nil || *drafts[0].Title != "Draft 2" {
			t.Errorf("drafts[0].Title = %v, want 'Draft 2'", drafts[0].Title)
		}
		if drafts[1].Title == nil || *drafts[1].Title != "Draft 3" {
			t.Errorf("drafts[1].Title = %v, want 'Draft 3'", drafts[1].Title)
		}
		if drafts[2].Title == nil || *drafts[2].Title != "Draft 1" {
			t.Errorf("drafts[2].Title = %v, want 'Draft 1'", drafts[2].Title)
		}

		// Verify the related entities are populated for the card view-model.
		// [Ja] 関連エンティティの情報が正しいことを確認する。
		if drafts[0].Page == nil || drafts[0].Page.Number != 2 {
			t.Errorf("drafts[0].Page.Number = %v, want 2", drafts[0].Page)
		}
		if drafts[0].Topic == nil || drafts[0].Topic.Name != "General" {
			t.Errorf("drafts[0].Topic.Name = %v, want 'General'", drafts[0].Topic)
		}
		if drafts[0].Topic.Space == nil || string(drafts[0].Topic.Space.Identifier) != "draft-sm-space" {
			t.Errorf("drafts[0].Topic.Space.Identifier = %v, want 'draft-sm-space'", drafts[0].Topic.Space)
		}
	})

	t.Run("LIMITが適用される", func(t *testing.T) {
		drafts, err := repo.ListBySpaceMember(context.Background(), spaceMemberID, spaceID, 2)
		if err != nil {
			t.Fatalf("ListBySpaceMember() error = %v", err)
		}
		if len(drafts) != 2 {
			t.Fatalf("ListBySpaceMember() returned %d drafts, want 2", len(drafts))
		}
	})

	t.Run("提案編集用の下書き (suggestion_page_id 付き) も含まれる", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(4).
			WithTitle("Suggestion Page").
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTitle("Suggestion Page").
			WithBody("body").
			WithBodyHTML("<p>body</p>").
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
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithSuggestionPageID(suggestionPageID).
			WithTitle("Suggestion Draft").
			WithModifiedAt(now.Add(1 * time.Hour)).
			Build()

		drafts, err := repo.ListBySpaceMember(context.Background(), spaceMemberID, spaceID, 20)
		if err != nil {
			t.Fatalf("ListBySpaceMember() error = %v", err)
		}
		found := false
		for _, d := range drafts {
			if d.Title != nil && *d.Title == "Suggestion Draft" {
				found = true
			}
		}
		if !found {
			t.Error("suggestion-edit draft should be included in the list")
		}
	})

	t.Run("別のスペースメンバーの下書きは含まれない", func(t *testing.T) {
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("draft-sm-other@example.com").
			WithAtname("draftsmother").
			Build()
		otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(otherUserID).
			Build()

		drafts, err := repo.ListBySpaceMember(context.Background(), otherMemberID, spaceID, 20)
		if err != nil {
			t.Fatalf("ListBySpaceMember() error = %v", err)
		}
		if len(drafts) != 0 {
			t.Errorf("ListBySpaceMember() returned %d drafts, want 0 for a member with no drafts", len(drafts))
		}
	})

	t.Run("論理削除されたページの下書きは除外される", func(t *testing.T) {
		discardedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(5).
			WithTitle("Discarded Page").
			WithDiscarded().
			Build()
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(discardedPageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("Draft on discarded page").
			WithModifiedAt(now.Add(2 * time.Hour)).
			Build()

		drafts, err := repo.ListBySpaceMember(context.Background(), spaceMemberID, spaceID, 20)
		if err != nil {
			t.Fatalf("ListBySpaceMember() error = %v", err)
		}
		for _, d := range drafts {
			if d.Title != nil && *d.Title == "Draft on discarded page" {
				t.Error("draft on a discarded page should be excluded")
			}
		}
	})
}

// Verifies the ON DELETE SET NULL contract on draft_pages that the Rails-side
// deletion paths rely on: deleting the referenced suggestion_pages /
// attachments row must null the reference while keeping the draft alive,
// because drafts are user work-in-progress and must not be deleted along.
//
// [Ja] Rails 側の削除経路が頼る draft_pages の ON DELETE SET NULL の契約を
// 検証する。参照先の suggestion_pages / attachments の行を削除したとき、
// 参照は NULL になり下書き自体は残ること。下書きはユーザーの書きかけ原稿で
// あり、巻き添えで消してはならない。
func TestDraftPageRepository_OnDeleteSetNull(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draft-ondelete@example.com").
		WithAtname("draft_ondelete").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draft-ondelete").
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

	t.Run("提案ページの行を直接DELETEすると下書きの参照がNULLになる", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Suggestion Page Draft").
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
			Build()

		draftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithSuggestionPageID(suggestionPageID).
			Build()

		// Delete the referenced suggestion_pages row directly (without going through application code)
		// [Ja] 参照先の suggestion_pages の行を直接削除 (アプリケーションコードを経由しない)
		_, err := tx.ExecContext(
			ctx,
			"DELETE FROM suggestion_pages WHERE id = $1 AND space_id = $2",
			string(suggestionPageID), string(spaceID),
		)
		if err != nil {
			t.Fatalf("DELETE suggestion_pages error = %v", err)
		}

		var suggestionPageIsNull bool
		if err := tx.QueryRowContext(
			ctx,
			"SELECT suggestion_page_id IS NULL FROM draft_pages WHERE id = $1 AND space_id = $2",
			string(draftPageID), string(spaceID),
		).Scan(&suggestionPageIsNull); err != nil {
			t.Fatalf("SELECT draft_pages.suggestion_page_id error = %v", err)
		}
		if !suggestionPageIsNull {
			t.Error("draft_pages.suggestion_page_id should be NULL after deleting the suggestion page")
		}
	})

	t.Run("添付ファイルの行を直接DELETEすると下書きの注目画像の参照がNULLになる", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Featured Image Draft").
			Build()

		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()

		draftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithFeaturedImageAttachmentID(attachmentID).
			Build()

		_, err := tx.ExecContext(
			ctx,
			"DELETE FROM attachments WHERE id = $1 AND space_id = $2",
			string(attachmentID), string(spaceID),
		)
		if err != nil {
			t.Fatalf("DELETE attachments error = %v", err)
		}

		var featuredImageIsNull bool
		if err := tx.QueryRowContext(
			ctx,
			"SELECT featured_image_attachment_id IS NULL FROM draft_pages WHERE id = $1 AND space_id = $2",
			string(draftPageID), string(spaceID),
		).Scan(&featuredImageIsNull); err != nil {
			t.Fatalf("SELECT draft_pages.featured_image_attachment_id error = %v", err)
		}
		if !featuredImageIsNull {
			t.Error("draft_pages.featured_image_attachment_id should be NULL after deleting the attachment")
		}

		// The draft itself must survive the attachment deletion.
		// [Ja] 下書き自体は添付ファイルの削除後も残ること。
		var draftCount int
		if err := tx.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM draft_pages WHERE id = $1 AND space_id = $2",
			string(draftPageID), string(spaceID),
		).Scan(&draftCount); err != nil {
			t.Fatalf("SELECT COUNT(*) FROM draft_pages error = %v", err)
		}
		if draftCount != 1 {
			t.Errorf("draft_pages count = %d, want 1 (draft must not be deleted)", draftCount)
		}
	})
}
