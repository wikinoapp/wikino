package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestAddSuggestionPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	pageRevisionRepo := repository.NewPageRevisionRepository(q)
	createValidator := validator.NewSuggestionPageCreateValidator(draftPageRepo, suggestionPageRepo)

	uc := NewAddSuggestionPageUsecase(db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo, pageRevisionRepo, createValidator)

	t.Run("正常系: 編集提案にページを追加できる", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("add-sp-ok").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("add-sp-ok@example.com").
			WithAtname("addspok").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()

		// 既存のページ（編集提案に含まれている）
		existingPageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("既存ページ").
			Build()
		existingPageRevisionID := createPageRevisionViaRepo(t, q, spaceID, spaceMemberID, existingPageID)

		// 編集提案を作成
		sugRepo := repository.NewSuggestionRepository(q)
		suggestion, err := sugRepo.Create(context.Background(), repository.CreateSuggestionInput{
			SpaceID:              spaceID,
			TopicID:              topicID,
			CreatedSpaceMemberID: spaceMemberID,
			Number:               1,
			Title:                "テスト提案",
			Body:                 "",
			BodyHTML:             "",
			Status:               model.SuggestionStatusOpen,
		})
		if err != nil {
			t.Fatalf("suggestion creation failed: %v", err)
		}

		// 既存のSuggestionPageを作成
		spRepo := repository.NewSuggestionPageRepository(q)
		_, err = spRepo.Create(context.Background(), repository.CreateSuggestionPageInput{
			SpaceID:        spaceID,
			SuggestionID:   suggestion.ID,
			PageID:         existingPageID,
			PageRevisionID: pageRevisionIDPtr(model.PageRevisionID(existingPageRevisionID)),
			Title:          strPtr("既存ページ"),
			Body:           "既存本文",
			BodyHTML:       "<p>既存本文</p>",
		})
		if err != nil {
			t.Fatalf("suggestion page creation failed: %v", err)
		}

		// 追加するページ
		newPageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("追加ページ").
			Build()
		createPageRevisionForTest(t, q, spaceID, spaceMemberID, newPageID)

		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(newPageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("追加ページ提案").
			WithBody("追加本文").
			WithBodyHTML("<p>追加本文</p>").
			Build()

		output, err := uc.Execute(context.Background(), AddSuggestionPageInput{
			SpaceIdentifier:  "add-sp-ok",
			SuggestionNumber: suggestion.Number,
			UserID:           userID,
			DraftPageIDs:     []model.DraftPageID{draftPageID},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil || output.Suggestion == nil {
			t.Fatal("output or Suggestion should not be nil")
		}

		// SuggestionPageが2つになったことを確認
		pages, err := suggestionPageRepo.ListBySuggestionID(context.Background(), suggestion.ID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(pages) != 2 {
			t.Errorf("SuggestionPages count = %d, want 2", len(pages))
		}

		// 追加されたSuggestionPageの内容を確認
		var addedPage *model.SuggestionPage
		for _, p := range pages {
			if p.PageID == newPageID {
				addedPage = p
				break
			}
		}
		if addedPage == nil {
			t.Fatal("added SuggestionPage not found")
		}
		if addedPage.Body != "追加本文" {
			t.Errorf("SuggestionPage.Body = %q, want %q", addedPage.Body, "追加本文")
		}

		// SuggestionPageRevisionが作成されたことを確認
		revisions, err := suggestionPageRevisionRepo.ListBySuggestionPageID(context.Background(), addedPage.ID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionPageID() error = %v", err)
		}
		if len(revisions) != 1 {
			t.Errorf("revisions count = %d, want 1", len(revisions))
		}

		// DraftPageのsuggestion_page_idが設定されたことを確認
		updatedDraft, err := draftPageRepo.FindByID(context.Background(), draftPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if updatedDraft.SuggestionPageID == nil {
			t.Fatal("DraftPage.SuggestionPageID should not be nil")
		}
		if *updatedDraft.SuggestionPageID != addedPage.ID {
			t.Errorf("DraftPage.SuggestionPageID = %v, want %v", *updatedDraft.SuggestionPageID, addedPage.ID)
		}
	})

	t.Run("正常系: 新規ページ（リビジョンなし）の下書きを編集提案に追加できる", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("add-sp-newpg").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("add-sp-newpg@example.com").
			WithAtname("addspnewpg").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()

		// 編集提案を作成
		sugRepo := repository.NewSuggestionRepository(q)
		suggestion, err := sugRepo.Create(context.Background(), repository.CreateSuggestionInput{
			SpaceID:              spaceID,
			TopicID:              topicID,
			CreatedSpaceMemberID: spaceMemberID,
			Number:               1,
			Title:                "新規ページ提案",
			Body:                 "",
			BodyHTML:             "",
			Status:               model.SuggestionStatusOpen,
		})
		if err != nil {
			t.Fatalf("suggestion creation failed: %v", err)
		}

		// 新規ページ（リビジョンなし）を作成
		newPageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("新規ページ").
			Build()
		// リビジョンは作成しない（新規ページ）

		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(newPageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("新規ページタイトル").
			WithBody("新規ページ本文").
			WithBodyHTML("<p>新規ページ本文</p>").
			Build()

		output, err := uc.Execute(context.Background(), AddSuggestionPageInput{
			SpaceIdentifier:  "add-sp-newpg",
			SuggestionNumber: suggestion.Number,
			UserID:           userID,
			DraftPageIDs:     []model.DraftPageID{draftPageID},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil || output.Suggestion == nil {
			t.Fatal("output or Suggestion should not be nil")
		}

		// SuggestionPageが作成されたことを確認
		pages, err := suggestionPageRepo.ListBySuggestionID(context.Background(), suggestion.ID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(pages) != 1 {
			t.Fatalf("SuggestionPages count = %d, want 1", len(pages))
		}

		addedPage := pages[0]
		if addedPage.PageRevisionID != nil {
			t.Errorf("SuggestionPage.PageRevisionID = %v, want nil", addedPage.PageRevisionID)
		}
		if addedPage.Body != "新規ページ本文" {
			t.Errorf("SuggestionPage.Body = %q, want %q", addedPage.Body, "新規ページ本文")
		}
	})

	t.Run("異常系: 存在しないスペースの場合AppErrorが返る", func(t *testing.T) {
		t.Parallel()

		_, err := uc.Execute(context.Background(), AddSuggestionPageInput{
			SpaceIdentifier:  "nonexistent-sp-add",
			SuggestionNumber: 1,
			UserID:           "dummy-user",
			DraftPageIDs:     []model.DraftPageID{"dummy"},
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %d, want %d", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: 存在しない編集提案の場合AppErrorが返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("add-sp-nosug").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("add-sp-nosug@example.com").
			WithAtname("addspnosug").
			Build()
		testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()

		_, err := uc.Execute(context.Background(), AddSuggestionPageInput{
			SpaceIdentifier:  "add-sp-nosug",
			SuggestionNumber: 999,
			UserID:           userID,
			DraftPageIDs:     []model.DraftPageID{"dummy"},
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %d, want %d", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: 非メンバーの場合AppError(Forbidden)が返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("add-sp-noauth").
			Build()
		ownerUserID := testutil.NewUserBuilderDB(t, db).
			WithEmail("add-sp-noauth-owner@example.com").
			WithAtname("addspnoauthowner").
			Build()
		ownerMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(ownerUserID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()

		sugRepo := repository.NewSuggestionRepository(q)
		suggestion, err := sugRepo.Create(context.Background(), repository.CreateSuggestionInput{
			SpaceID:              spaceID,
			TopicID:              topicID,
			CreatedSpaceMemberID: ownerMemberID,
			Number:               1,
			Title:                "テスト提案",
			Body:                 "",
			BodyHTML:             "",
			Status:               model.SuggestionStatusOpen,
		})
		if err != nil {
			t.Fatalf("suggestion creation failed: %v", err)
		}

		nonMemberID := testutil.NewUserBuilderDB(t, db).
			WithEmail("add-sp-noauth-non@example.com").
			WithAtname("addspnoauthnon").
			Build()

		_, err = uc.Execute(context.Background(), AddSuggestionPageInput{
			SpaceIdentifier:  "add-sp-noauth",
			SuggestionNumber: suggestion.Number,
			UserID:           nonMemberID,
			DraftPageIDs:     []model.DraftPageID{"dummy"},
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError, got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("Code = %d, want %d", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: 下書きページ未選択の場合ValidationErrorが返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("add-sp-nodp").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("add-sp-nodp@example.com").
			WithAtname("addspnodp").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()

		sugRepo := repository.NewSuggestionRepository(q)
		suggestion, err := sugRepo.Create(context.Background(), repository.CreateSuggestionInput{
			SpaceID:              spaceID,
			TopicID:              topicID,
			CreatedSpaceMemberID: spaceMemberID,
			Number:               1,
			Title:                "テスト提案",
			Body:                 "",
			BodyHTML:             "",
			Status:               model.SuggestionStatusOpen,
		})
		if err != nil {
			t.Fatalf("suggestion creation failed: %v", err)
		}

		_, err = uc.Execute(context.Background(), AddSuggestionPageInput{
			SpaceIdentifier:  "add-sp-nodp",
			SuggestionNumber: suggestion.Number,
			UserID:           userID,
			DraftPageIDs:     []model.DraftPageID{},
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if !ve.HasFieldError("draft_page_ids") {
			t.Error("expected draft_page_ids field error")
		}
	})

	t.Run("異常系: 重複ページの場合ValidationErrorが返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("add-sp-dup").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("add-sp-dup@example.com").
			WithAtname("addspdup").
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
			WithTitle("対象ページ").
			Build()
		pageRevisionID := createPageRevisionViaRepo(t, q, spaceID, spaceMemberID, pageID)

		sugRepo := repository.NewSuggestionRepository(q)
		suggestion, err := sugRepo.Create(context.Background(), repository.CreateSuggestionInput{
			SpaceID:              spaceID,
			TopicID:              topicID,
			CreatedSpaceMemberID: spaceMemberID,
			Number:               1,
			Title:                "テスト提案",
			Body:                 "",
			BodyHTML:             "",
			Status:               model.SuggestionStatusOpen,
		})
		if err != nil {
			t.Fatalf("suggestion creation failed: %v", err)
		}

		// 既存のSuggestionPage
		spRepo := repository.NewSuggestionPageRepository(q)
		_, err = spRepo.Create(context.Background(), repository.CreateSuggestionPageInput{
			SpaceID:        spaceID,
			SuggestionID:   suggestion.ID,
			PageID:         pageID,
			PageRevisionID: pageRevisionIDPtr(model.PageRevisionID(pageRevisionID)),
			Title:          strPtr("対象ページ"),
			Body:           "本文",
			BodyHTML:       "<p>本文</p>",
		})
		if err != nil {
			t.Fatalf("suggestion page creation failed: %v", err)
		}

		// 同じページの下書き
		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("対象ページ").
			WithBody("本文").
			WithBodyHTML("<p>本文</p>").
			Build()

		_, err = uc.Execute(context.Background(), AddSuggestionPageInput{
			SpaceIdentifier:  "add-sp-dup",
			SuggestionNumber: suggestion.Number,
			UserID:           userID,
			DraftPageIDs:     []model.DraftPageID{draftPageID},
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if !ve.HasFieldError("draft_page_ids") {
			t.Error("expected draft_page_ids field error")
		}
	})
}

func pageRevisionIDPtr(id model.PageRevisionID) *model.PageRevisionID {
	return &id
}
