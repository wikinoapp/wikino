package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestSuggestionPageRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionPageRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-create@example.com").
		WithAtname("sp_create").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-create-space").
		WithName("SP Create Space").
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
		WithTitle("テストページ").
		Build()

	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		WithTitle("テストページ").
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("テスト提案").
		Build()

	t.Run("タイトルありで編集提案ページを作成できる", func(t *testing.T) {
		title := "提案ページタイトル"
		sp, err := repo.Create(ctx, CreateSuggestionPageInput{
			SpaceID:        spaceID,
			SuggestionID:   suggestionID,
			PageID:         pageID,
			PageRevisionID: &pageRevisionID,
			Title:          &title,
			Body:           "提案ページ本文",
			BodyHTML:       "<p>提案ページ本文</p>",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		if sp == nil {
			t.Fatal("Create()がnilを返した")
		}
		if sp.ID == "" {
			t.Error("sp.IDが空")
		}
		if sp.SpaceID != spaceID {
			t.Errorf("sp.SpaceID = %v、期待値 = %v", sp.SpaceID, spaceID)
		}
		if sp.SuggestionID != suggestionID {
			t.Errorf("sp.SuggestionID = %v、期待値 = %v", sp.SuggestionID, suggestionID)
		}
		if sp.PageID != pageID {
			t.Errorf("sp.PageID = %v、期待値 = %v", sp.PageID, pageID)
		}
		if sp.PageRevisionID == nil || *sp.PageRevisionID != pageRevisionID {
			t.Errorf("sp.PageRevisionID = %v、期待値 = %v", sp.PageRevisionID, pageRevisionID)
		}
		if sp.Title == nil || *sp.Title != "提案ページタイトル" {
			t.Errorf("sp.Title = %v、期待値 = 提案ページタイトル", sp.Title)
		}
		if sp.Body != "提案ページ本文" {
			t.Errorf("sp.Body = %v、期待値 = 提案ページ本文", sp.Body)
		}
		if sp.BodyHTML != "<p>提案ページ本文</p>" {
			t.Errorf("sp.BodyHTML = %v、期待値 = <p>提案ページ本文</p>", sp.BodyHTML)
		}
	})

	t.Run("PageRevisionIDなしで編集提案ページを作成できる", func(t *testing.T) {
		// 別のページを作成してユニーク制約を回避
		pageID3 := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(3).
			WithTitle("新規ページ").
			WithUnpublished().
			Build()

		title := "新規ページ提案"
		sp, err := repo.Create(ctx, CreateSuggestionPageInput{
			SpaceID:        spaceID,
			SuggestionID:   suggestionID,
			PageID:         pageID3,
			PageRevisionID: nil,
			Title:          &title,
			Body:           "新規ページ本文",
			BodyHTML:       "<p>新規ページ本文</p>",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		if sp == nil {
			t.Fatal("Create()がnilを返した")
		}
		if sp.PageRevisionID != nil {
			t.Errorf("sp.PageRevisionID = %v、期待値 = nil", sp.PageRevisionID)
		}
		if sp.Title == nil || *sp.Title != "新規ページ提案" {
			t.Errorf("sp.Title = %v、期待値 = 新規ページ提案", sp.Title)
		}
		if sp.Body != "新規ページ本文" {
			t.Errorf("sp.Body = %v、期待値 = 新規ページ本文", sp.Body)
		}
	})

	t.Run("タイトルなしで編集提案ページを作成できる", func(t *testing.T) {
		// 別のページを作成してユニーク制約を回避
		pageID2 := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("テストページ2").
			Build()

		pageRevisionID2 := testutil.NewPageRevisionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID2).
			WithTitle("テストページ2").
			Build()

		sp, err := repo.Create(ctx, CreateSuggestionPageInput{
			SpaceID:        spaceID,
			SuggestionID:   suggestionID,
			PageID:         pageID2,
			PageRevisionID: &pageRevisionID2,
			Title:          nil,
			Body:           "タイトルなし本文",
			BodyHTML:       "<p>タイトルなし本文</p>",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		if sp.Title != nil {
			t.Errorf("sp.Title = %v、期待値 = nil", sp.Title)
		}
	})
}

func TestSuggestionPageRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionPageRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-find@example.com").
		WithAtname("sp_find").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-find-space").
		WithName("SP Find Space").
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
		Build()

	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
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
		WithTitle("検索テスト提案ページ").
		Build()

	t.Run("IDで編集提案ページを取得できる", func(t *testing.T) {
		sp, err := repo.FindByID(ctx, suggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if sp == nil {
			t.Fatal("FindByID()がnilを返した")
		}
		if sp.ID != suggestionPageID {
			t.Errorf("sp.ID = %v、期待値 = %v", sp.ID, suggestionPageID)
		}
		if sp.Title == nil || *sp.Title != "検索テスト提案ページ" {
			t.Errorf("sp.Title = %v、期待値 = 検索テスト提案ページ", sp.Title)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		sp, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000", spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if sp != nil {
			t.Errorf("FindByID() = %v、期待値 = nil", sp)
		}
	})

	t.Run("異なるスペースIDではnilを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("sp-find-other").
			WithName("Other Space").
			Build()

		sp, err := repo.FindByID(ctx, suggestionPageID, otherSpaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if sp != nil {
			t.Errorf("FindByID() = %v、期待値 = nil", sp)
		}
	})
}

func TestSuggestionPageRepository_ListBySuggestionID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionPageRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-list@example.com").
		WithAtname("sp_list").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-list-space").
		WithName("SP List Space").
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

	pageID1 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("ページ1").
		Build()

	pageRevisionID1 := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID1).
		Build()

	pageID2 := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("ページ2").
		Build()

	pageRevisionID2 := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID2).
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID1).
		WithPageRevisionID(pageRevisionID1).
		WithTitle("提案ページ1").
		Build()

	testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID2).
		WithPageRevisionID(pageRevisionID2).
		WithTitle("提案ページ2").
		Build()

	t.Run("編集提案に紐づくページ一覧を取得できる", func(t *testing.T) {
		pages, err := repo.ListBySuggestionID(ctx, suggestionID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID()のエラー = %v", err)
		}
		if len(pages) != 2 {
			t.Fatalf("len(pages) = %v、期待値 = 2", len(pages))
		}
	})

	t.Run("該当なしの場合は空のスライスを返す", func(t *testing.T) {
		otherSuggestionID := testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithTitle("空の提案").
			Build()

		pages, err := repo.ListBySuggestionID(ctx, otherSuggestionID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID()のエラー = %v", err)
		}
		if len(pages) != 0 {
			t.Errorf("len(pages) = %v、期待値 = 0", len(pages))
		}
	})
}

func TestSuggestionPageRepository_UpdateContent(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionPageRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-update@example.com").
		WithAtname("sp_update").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-update-space").
		WithName("SP Update Space").
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
		Build()

	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	t.Run("コンテンツを更新できる", func(t *testing.T) {
		suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			WithTitle("更新前タイトル").
			WithBody("更新前本文").
			WithBodyHTML("<p>更新前本文</p>").
			Build()

		newTitle := "更新後タイトル"
		sp, err := repo.UpdateContent(ctx, UpdateSuggestionPageContentInput{
			ID:       suggestionPageID,
			SpaceID:  spaceID,
			Title:    &newTitle,
			Body:     "更新後本文",
			BodyHTML: "<p>更新後本文</p>",
		})
		if err != nil {
			t.Fatalf("UpdateContent()のエラー = %v", err)
		}
		if sp == nil {
			t.Fatal("UpdateContent()がnilを返した")
		}
		if sp.Title == nil || *sp.Title != "更新後タイトル" {
			t.Errorf("sp.Title = %v、期待値 = 更新後タイトル", sp.Title)
		}
		if sp.Body != "更新後本文" {
			t.Errorf("sp.Body = %v、期待値 = 更新後本文", sp.Body)
		}
		if sp.BodyHTML != "<p>更新後本文</p>" {
			t.Errorf("sp.BodyHTML = %v、期待値 = <p>更新後本文</p>", sp.BodyHTML)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		title := "更新タイトル"
		sp, err := repo.UpdateContent(ctx, UpdateSuggestionPageContentInput{
			ID:       "00000000-0000-0000-0000-000000000000",
			SpaceID:  spaceID,
			Title:    &title,
			Body:     "更新本文",
			BodyHTML: "<p>更新本文</p>",
		})
		if err != nil {
			t.Fatalf("UpdateContent()のエラー = %v", err)
		}
		if sp != nil {
			t.Errorf("UpdateContent() = %v、期待値 = nil", sp)
		}
	})
}

func TestSuggestionPageRepository_Delete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionPageRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-delete@example.com").
		WithAtname("sp_delete").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-delete-space").
		WithName("SP Delete Space").
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
		Build()

	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		Build()

	t.Run("編集提案ページを削除できる", func(t *testing.T) {
		suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			WithTitle("削除テスト").
			Build()

		err := repo.Delete(ctx, suggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("Delete()のエラー = %v", err)
		}

		sp, err := repo.FindByID(ctx, suggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if sp != nil {
			t.Errorf("FindByID() = %v、期待値 = nil (削除済み)", sp)
		}
	})

	t.Run("存在しないIDで削除してもエラーにならない", func(t *testing.T) {
		err := repo.Delete(ctx, "00000000-0000-0000-0000-000000000000", spaceID)
		if err != nil {
			t.Fatalf("Delete()のエラー = %v", err)
		}
	})

	t.Run("異なるスペースIDでは削除されない", func(t *testing.T) {
		pageID2 := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("削除テストページ2").
			Build()

		pageRevisionID2 := testutil.NewPageRevisionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID2).
			Build()

		suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID2).
			WithPageRevisionID(pageRevisionID2).
			WithTitle("別スペース削除テスト").
			Build()

		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("sp-delete-other").
			WithName("Other Space").
			Build()

		err := repo.Delete(ctx, suggestionPageID, otherSpaceID)
		if err != nil {
			t.Fatalf("Delete()のエラー = %v", err)
		}

		sp, err := repo.FindByID(ctx, suggestionPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if sp == nil {
			t.Error("FindByID() = nil、期待値 = nilではない (別スペースの指定で削除されている)")
		}
	})
}

func TestSuggestionPageRepository_ExistsByPageIDAndOpenStatus(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewSuggestionPageRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-exists-open@example.com").
		WithAtname("sp_exists_open").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-exists-open-space").
		WithName("SP Exists Open Space").
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

	t.Run("オープンな編集提案がページを参照している場合trueを返す", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(10).
			WithTitle("オープン参照ページ").
			Build()

		pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID).
			Build()

		suggestionID := testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			WithTitle("オープン提案").
			Build()

		testutil.NewSuggestionPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		exists, err := repo.ExistsByPageIDAndOpenStatus(ctx, pageID, spaceID)
		if err != nil {
			t.Fatalf("ExistsByPageIDAndOpenStatus()のエラー = %v", err)
		}
		if !exists {
			t.Error("ExistsByPageIDAndOpenStatus() = false、期待値 = true")
		}
	})

	t.Run("クローズ済みの編集提案のみの場合falseを返す", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(11).
			WithTitle("クローズ参照ページ").
			Build()

		pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID).
			Build()

		suggestionID := testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusClosed).
			WithTitle("クローズ提案").
			Build()

		testutil.NewSuggestionPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		exists, err := repo.ExistsByPageIDAndOpenStatus(ctx, pageID, spaceID)
		if err != nil {
			t.Fatalf("ExistsByPageIDAndOpenStatus()のエラー = %v", err)
		}
		if exists {
			t.Error("ExistsByPageIDAndOpenStatus() = true、期待値 = false")
		}
	})

	t.Run("編集提案がない場合falseを返す", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(12).
			WithTitle("参照なしページ").
			Build()

		exists, err := repo.ExistsByPageIDAndOpenStatus(ctx, pageID, spaceID)
		if err != nil {
			t.Fatalf("ExistsByPageIDAndOpenStatus()のエラー = %v", err)
		}
		if exists {
			t.Error("ExistsByPageIDAndOpenStatus() = true、期待値 = false")
		}
	})

	t.Run("異なるスペースIDではfalseを返す", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(13).
			WithTitle("別スペーステストページ").
			Build()

		pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithPageID(pageID).
			Build()

		suggestionID := testutil.NewSuggestionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			WithTitle("別スペーステスト提案").
			Build()

		testutil.NewSuggestionPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("sp-exists-other").
			WithName("Other Space").
			Build()

		exists, err := repo.ExistsByPageIDAndOpenStatus(ctx, pageID, otherSpaceID)
		if err != nil {
			t.Fatalf("ExistsByPageIDAndOpenStatus()のエラー = %v", err)
		}
		if exists {
			t.Error("ExistsByPageIDAndOpenStatus() = true、期待値 = false")
		}
	})
}

// Rails側の削除経路が頼るsuggestion_pagesのON DELETEの契約を検証
// する。pagesの行の削除はsuggestion_pagesに連鎖し、page_revisions /
// attachmentsの行の削除は参照をNULLにするだけであること。
func TestSuggestionPageRepository_OnDeleteContract(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sp-ondelete@example.com").
		WithAtname("sp_ondelete").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sp-ondelete").
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

	t.Run("ページの行を直接DELETEすると提案ページも消える", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Cascade Page").
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

		// 親のpagesの行を直接削除 (アプリケーションコードを経由しない)
		_, err := tx.ExecContext(
			ctx,
			"DELETE FROM pages WHERE id = $1 AND space_id = $2",
			string(pageID), string(spaceID),
		)
		if err != nil {
			t.Fatalf("DELETE pagesのエラー = %v", err)
		}

		var count int
		if err := tx.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM suggestion_pages WHERE id = $1 AND space_id = $2",
			string(suggestionPageID), string(spaceID),
		).Scan(&count); err != nil {
			t.Fatalf("SELECT COUNT(*) FROM suggestion_pagesのエラー = %v", err)
		}
		if count != 0 {
			t.Errorf("suggestion_pagesの件数 = %d、期待値 = 0 (カスケード削除されていない)", count)
		}

		// 提案自体は残ること (提案ページのエントリだけが消える)。
		var suggestionCount int
		if err := tx.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM suggestions WHERE id = $1 AND space_id = $2",
			string(suggestionID), string(spaceID),
		).Scan(&suggestionCount); err != nil {
			t.Fatalf("SELECT COUNT(*) FROM suggestionsのエラー = %v", err)
		}
		if suggestionCount != 1 {
			t.Errorf("suggestionsの件数 = %d、期待値 = 1 (編集提案が削除されている)", suggestionCount)
		}
	})

	t.Run("ページリビジョンの行を直接DELETEすると参照がNULLになる", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Revision Page").
			Build()

		pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTitle("Revision Page").
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

		_, err := tx.ExecContext(
			ctx,
			"DELETE FROM page_revisions WHERE id = $1 AND space_id = $2",
			string(pageRevisionID), string(spaceID),
		)
		if err != nil {
			t.Fatalf("DELETE page_revisionsのエラー = %v", err)
		}

		var pageRevisionIsNull bool
		if err := tx.QueryRowContext(
			ctx,
			"SELECT page_revision_id IS NULL FROM suggestion_pages WHERE id = $1 AND space_id = $2",
			string(suggestionPageID), string(spaceID),
		).Scan(&pageRevisionIsNull); err != nil {
			t.Fatalf("SELECT suggestion_pages.page_revision_idのエラー = %v", err)
		}
		if !pageRevisionIsNull {
			t.Error("ページのリビジョンの削除後もsuggestion_pages.page_revision_idがNULLになっていない")
		}
	})

	t.Run("添付ファイルの行を直接DELETEすると注目画像の参照がNULLになる", func(t *testing.T) {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(3).
			WithTitle("Featured Image Page").
			Build()

		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
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
			WithFeaturedImageAttachmentID(attachmentID).
			Build()

		_, err := tx.ExecContext(
			ctx,
			"DELETE FROM attachments WHERE id = $1 AND space_id = $2",
			string(attachmentID), string(spaceID),
		)
		if err != nil {
			t.Fatalf("DELETE attachmentsのエラー = %v", err)
		}

		var featuredImageIsNull bool
		if err := tx.QueryRowContext(
			ctx,
			"SELECT featured_image_attachment_id IS NULL FROM suggestion_pages WHERE id = $1 AND space_id = $2",
			string(suggestionPageID), string(spaceID),
		).Scan(&featuredImageIsNull); err != nil {
			t.Fatalf("SELECT suggestion_pages.featured_image_attachment_idのエラー = %v", err)
		}
		if !featuredImageIsNull {
			t.Error("添付ファイルの削除後もsuggestion_pages.featured_image_attachment_idがNULLになっていない")
		}
	})
}
