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

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	suggestionRepo := repository.NewSuggestionRepository(q)

	draftPageRepo := repository.NewDraftPageRepository(q)
	uc := NewCloseSuggestionUsecase(db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, draftPageRepo)

	t.Run("正常系: スペースオーナーがオープンの編集提案をクローズできる", func(t *testing.T) {
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
		pageID := testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Close Test Page").
			Build()
		suggestionID := testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()
		suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithSuggestionID(suggestionID).
			WithPageID(pageID).
			WithTitle("提案タイトル").
			WithBody("提案本文").
			Build()

		// 編集提案ページにリンクされた下書きを作成
		draftPageID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithBody("Draft body").
			WithSuggestionPageID(suggestionPageID).
			Build()

		output, err := uc.Execute(context.Background(), CloseSuggestionInput{
			SpaceIdentifier:  "close-sugg-1",
			SuggestionNumber: 1,
			UserID:           userID,
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

		// 下書きのsuggestion_page_idがクリアされていることを確認
		dp, err := draftPageRepo.FindByID(context.Background(), draftPageID, spaceID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if dp != nil && dp.SuggestionPageID != nil {
			t.Error("DraftPage.SuggestionPageID should be nil after close")
		}
	})

	t.Run("正常系: クローズ済みの編集提案はべき等に成功する", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("close-sugg-idem").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("close-sugg-idem@example.com").
			WithAtname("closesuggidem").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusClosed).
			Build()

		output, err := uc.Execute(context.Background(), CloseSuggestionInput{
			SpaceIdentifier:  "close-sugg-idem",
			SuggestionNumber: 1,
			UserID:           userID,
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

	t.Run("異常系: 存在しないスペースで AppErrCodeResourceNotFound が返る", func(t *testing.T) {
		t.Parallel()

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("close-sugg-nosp@example.com").
			WithAtname("closesuggnosp").
			Build()

		_, err := uc.Execute(context.Background(), CloseSuggestionInput{
			SpaceIdentifier:  "nonexistent-space",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error code = %d, want %d", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: スペースメンバーでないユーザーは AppErrCodeForbidden が返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("close-sugg-forbid").
			Build()
		ownerID := testutil.NewUserBuilderDB(t, db).
			WithEmail("close-sugg-forbid-owner@example.com").
			WithAtname("closesuggforbidowner").
			Build()
		nonMemberID := testutil.NewUserBuilderDB(t, db).
			WithEmail("close-sugg-forbid-nm@example.com").
			WithAtname("closesuggforbidnm").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(ownerID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		_, err := uc.Execute(context.Background(), CloseSuggestionInput{
			SpaceIdentifier:  "close-sugg-forbid",
			SuggestionNumber: 1,
			UserID:           nonMemberID,
		})
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("error code = %d, want %d", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: 権限のない一般メンバーは AppErrCodeForbidden が返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("close-sugg-noperm").
			Build()
		ownerID := testutil.NewUserBuilderDB(t, db).
			WithEmail("close-sugg-noperm-owner@example.com").
			WithAtname("closesuggnopermowner").
			Build()
		memberID := testutil.NewUserBuilderDB(t, db).
			WithEmail("close-sugg-noperm-member@example.com").
			WithAtname("closesuggnopermmember").
			Build()
		ownerSmID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(ownerID).
			Build()
		testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(memberID).
			WithRole(int32(model.SpaceMemberRoleMember)).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		// トピックメンバーを作成しない → トピックのゲストとして権限なし
		// 提案はオーナーが作成（一般メンバーは作成者ではない）
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(ownerSmID).
			WithStatus(model.SuggestionStatusOpen).
			Build()

		_, err := uc.Execute(context.Background(), CloseSuggestionInput{
			SpaceIdentifier:  "close-sugg-noperm",
			SuggestionNumber: 1,
			UserID:           memberID,
		})
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("error code = %d, want %d", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("異常系: 反映済みの編集提案は AppErrCodeConflict が返る", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("close-sugg-applied").
			Build()
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("close-sugg-applied@example.com").
			WithAtname("closesuggapplied").
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithUserID(userID).
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewSuggestionBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithCreatedSpaceMemberID(spaceMemberID).
			WithStatus(model.SuggestionStatusApplied).
			Build()

		_, err := uc.Execute(context.Background(), CloseSuggestionInput{
			SpaceIdentifier:  "close-sugg-applied",
			SuggestionNumber: 1,
			UserID:           userID,
		})
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeConflict {
			t.Errorf("error code = %d, want %d", ae.Code, model.AppErrCodeConflict)
		}
	})
}
