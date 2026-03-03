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
