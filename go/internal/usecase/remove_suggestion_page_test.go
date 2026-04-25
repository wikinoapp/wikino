package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestRemoveSuggestionPageUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)
	suggestionPageRepo := repository.NewSuggestionPageRepository(q)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)

	uc := NewRemoveSuggestionPageUsecase(
		db, spaceRepo, spaceMemberRepo, topicMemberRepo,
		suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo,
	)

	t.Run("正常系: 編集提案ページが削除されDraftPageのsuggestion_page_idがクリアされる", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("rm-sp-uc-ok").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("rm-sp-uc-ok@example.com").
			WithAtname("rmspucok").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		// 2つのページを作成（削除には2つ以上必要）
		pageID1 := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page 1").
			Build()
		pageRevisionID1 := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID1).
			WithSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID1 := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID1).
			WithPageRevisionID(pageRevisionID1).
			WithBody("提案ページ1").
			Build()

		pageID2 := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Page 2").
			Build()
		pageRevisionID2 := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID2).
			WithSpaceMemberID(spaceMemberID).
			Build()
		testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID2).
			WithPageRevisionID(pageRevisionID2).
			WithBody("提案ページ2").
			Build()

		// 削除対象のSuggestionPageに紐づくDraftPageを作成
		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID1).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithSuggestionPageID(suggestionPageID1).
			WithBody("下書き本文").
			Build()

		output, err := uc.Execute(context.Background(), RemoveSuggestionPageInput{
			SpaceIdentifier:  "rm-sp-uc-ok",
			SuggestionNumber: 1,
			SuggestionPageID: suggestionPageID1,
			UserID:           userID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}

		// SuggestionPageが削除されたことを確認
		deletedSP, err := suggestionPageRepo.FindByID(context.Background(), suggestionPageID1, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if deletedSP != nil {
			t.Error("SuggestionPageが削除されていません")
		}

		// SuggestionPageRevisionが削除されたことを確認
		revisions, err := suggestionPageRevisionRepo.ListBySuggestionPageID(context.Background(), suggestionPageID1, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionPageID() error = %v", err)
		}
		if len(revisions) != 0 {
			t.Errorf("SuggestionPageRevisionが残っています: %d件", len(revisions))
		}

		// DraftPageのsuggestion_page_idがクリアされたことを確認
		dp, err := draftPageRepo.FindByID(context.Background(), draftPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if dp == nil {
			t.Fatal("DraftPageが見つかりません")
		}
		if dp.SuggestionPageID != nil {
			t.Errorf("DraftPage.SuggestionPageID = %v, want nil", *dp.SuggestionPageID)
		}

		// 残りのSuggestionPageが存在することを確認
		remainingPages, err := suggestionPageRepo.ListBySuggestionID(context.Background(), suggestionID, spaceID)
		if err != nil {
			t.Fatalf("ListBySuggestionID() error = %v", err)
		}
		if len(remainingPages) != 1 {
			t.Errorf("残りの編集提案ページ数 = %d, want 1", len(remainingPages))
		}
	})

	t.Run("異常系: 最後の1ページは削除できない", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("rm-sp-uc-last").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("rm-sp-uc-last@example.com").
			WithAtname("rmsplast").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		_, err := uc.Execute(context.Background(), RemoveSuggestionPageInput{
			SpaceIdentifier:  "rm-sp-uc-last",
			SuggestionNumber: 1,
			SuggestionPageID: suggestionPageID,
			UserID:           userID,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeConflict {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeConflict)
		}
	})

	t.Run("異常系: スペースメンバーでない場合はForbiddenが返る", func(t *testing.T) {
		t.Parallel()

		nonMemberID := testutil.NewUserBuilderDB(t, db).
			WithEmail("rm-sp-uc-nonmem@example.com").
			WithAtname("rmspnonmem").
			Build()
		ownerID := testutil.NewUserBuilderDB(t, db).
			WithEmail("rm-sp-uc-owner@example.com").
			WithAtname("rmspowner").
			Build()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("rm-sp-uc-forbid").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(ownerID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		_, err := uc.Execute(context.Background(), RemoveSuggestionPageInput{
			SpaceIdentifier:  "rm-sp-uc-forbid",
			SuggestionNumber: 1,
			SuggestionPageID: suggestionPageID,
			UserID:           nonMemberID,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: クローズ済みの編集提案はForbiddenが返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("rm-sp-uc-closed").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("rm-sp-uc-closed@example.com").
			WithAtname("rmspclosed").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("Topic").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusClosed).
			Build()

		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			Build()
		pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithPageRevisionID(pageRevisionID).
			Build()

		_, err := uc.Execute(context.Background(), RemoveSuggestionPageInput{
			SpaceIdentifier:  "rm-sp-uc-closed",
			SuggestionNumber: 1,
			SuggestionPageID: suggestionPageID,
			UserID:           userID,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: 存在しないスペースの場合はResourceNotFoundが返る", func(t *testing.T) {
		t.Parallel()

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("rm-sp-uc-nosp@example.com").
			WithAtname("rmspnosp").
			Build()

		_, err := uc.Execute(context.Background(), RemoveSuggestionPageInput{
			SpaceIdentifier:  "nonexistent-space",
			SuggestionNumber: 1,
			SuggestionPageID: "test-sp-id",
			UserID:           userID,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %T: %v", err, err)
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})
}
